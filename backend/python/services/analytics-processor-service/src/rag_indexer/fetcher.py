import asyncio
import logging
from typing import Optional, Tuple, Dict, Any, List
from urllib.parse import urlparse

import aiohttp
import aioboto3
from botocore.exceptions import ClientError

from .config import settings
from . import metrics

log = logging.getLogger(__name__)


class FetchError(Exception):
    pass


async def fetch_content(uri: str, etag: Optional[str] = None, metadata: Optional[Dict[str, Any]] = None) -> Tuple[Optional[str], Optional[str]]:
    """Fetch content from an allowed URI.

    Returns (text, new_etag). If server replies 304/NotModified, returns (None, etag).
    Raises FetchError on disallowed scheme or network errors.
    """

    metadata = metadata or {}
    parsed = urlparse(uri)
    if parsed.scheme not in settings.allowed_schemes:
        raise FetchError(f"scheme not allowed: {parsed.scheme}")

    if parsed.scheme in {"http", "https"}:
        if metadata.get("graphql_query"):
            return await _fetch_graphql(uri, metadata)
        if metadata.get("rest_paginate") or metadata.get("rest_page_param"):
            return await _fetch_rest_paginated(uri, metadata)
        return await _fetch_http(uri, etag)
    if parsed.scheme == "s3":
        return await _fetch_s3(parsed, etag)

    raise FetchError(f"unsupported scheme: {parsed.scheme}")


async def _fetch_http(uri: str, etag: Optional[str]) -> Tuple[Optional[str], Optional[str]]:
    timeout = aiohttp.ClientTimeout(total=settings.fetch_timeout_seconds)
    headers = {}
    if etag:
        headers["If-None-Match"] = etag

    async with aiohttp.ClientSession(timeout=timeout) as session:
        try:
            async with session.get(uri, headers=headers) as resp:
                if resp.status == 304:
                    return None, etag
                if resp.status != 200:
                    raise FetchError(f"unexpected status {resp.status}")
                text = await resp.text()
                return text, resp.headers.get("ETag", etag)
        except asyncio.TimeoutError as exc:
            raise FetchError("fetch timed out") from exc
        except aiohttp.ClientError as exc:
            raise FetchError(f"fetch failed: {exc}") from exc


async def _fetch_rest_paginated(uri: str, meta: Dict[str, Any]) -> Tuple[Optional[str], Optional[str]]:
    page_param = str(meta.get("rest_page_param", "page"))
    page_start = int(meta.get("rest_page_start", 1))
    page_size = meta.get("rest_page_size")
    max_pages = int(meta.get("rest_max_pages", 5))
    headers = meta.get("rest_headers") or {}
    bodies: List[str] = []
    timeout = aiohttp.ClientTimeout(total=settings.fetch_timeout_seconds)
    params = meta.get("rest_params") or {}

    async with aiohttp.ClientSession(timeout=timeout) as session:
        page = page_start
        for _ in range(max_pages):
            req_params = dict(params)
            req_params[page_param] = page
            if page_size:
                req_params[meta.get("rest_page_size_param", "page_size")] = page_size
            metrics.REST_FETCH_ATTEMPTS.labels(kind="rest").inc()
            try:
                async with session.get(uri, params=req_params, headers=headers) as resp:
                    start = asyncio.get_event_loop().time()
                    if resp.status != 200:
                        metrics.REST_FETCH_FAILED.labels(kind="rest", reason=str(resp.status)).inc()
                        raise FetchError(f"unexpected status {resp.status}")
                    text = await resp.text()
                    metrics.REST_FETCH_LATENCY.labels(kind="rest").observe(asyncio.get_event_loop().time() - start)
                    # simple stop condition: empty body or content-length zero
                    if not text or len(text.strip()) == 0:
                        break
                    bodies.append(text)
            except asyncio.TimeoutError as exc:
                metrics.REST_FETCH_FAILED.labels(kind="rest", reason="timeout").inc()
                raise FetchError("rest fetch timed out") from exc
            except aiohttp.ClientError as exc:
                metrics.REST_FETCH_FAILED.labels(kind="rest", reason="client_error").inc()
                raise FetchError(f"rest fetch failed: {exc}") from exc
            page += 1

    combined = "\n".join(bodies)
    return combined, None


async def _fetch_graphql(uri: str, meta: Dict[str, Any]) -> Tuple[Optional[str], Optional[str]]:
    query = meta.get("graphql_query")
    if not query:
        raise FetchError("graphql_query required")
    variables = meta.get("graphql_variables") or {}
    headers = {"Content-Type": "application/json"}
    headers.update(meta.get("graphql_headers") or {})
    timeout = aiohttp.ClientTimeout(total=settings.fetch_timeout_seconds)
    metrics.REST_FETCH_ATTEMPTS.labels(kind="graphql").inc()
    async with aiohttp.ClientSession(timeout=timeout) as session:
        try:
            start = asyncio.get_event_loop().time()
            async with session.post(uri, json={"query": query, "variables": variables}, headers=headers) as resp:
                if resp.status != 200:
                    metrics.REST_FETCH_FAILED.labels(kind="graphql", reason=str(resp.status)).inc()
                    raise FetchError(f"graphql status {resp.status}")
                text = await resp.text()
                metrics.REST_FETCH_LATENCY.labels(kind="graphql").observe(asyncio.get_event_loop().time() - start)
                return text, None
        except asyncio.TimeoutError as exc:
            metrics.REST_FETCH_FAILED.labels(kind="graphql", reason="timeout").inc()
            raise FetchError("graphql fetch timed out") from exc
        except aiohttp.ClientError as exc:
            metrics.REST_FETCH_FAILED.labels(kind="graphql", reason="client_error").inc()
            raise FetchError(f"graphql fetch failed: {exc}") from exc


async def _fetch_s3(parsed, etag: Optional[str]) -> Tuple[Optional[str], Optional[str]]:
    bucket = parsed.netloc
    key = parsed.path.lstrip("/")
    if not bucket or not key:
        raise FetchError("invalid s3 uri")

    session = aioboto3.Session()
    params = {
        "endpoint_url": settings.s3_endpoint_url or None,
        "aws_access_key_id": settings.s3_access_key_id or None,
        "aws_secret_access_key": settings.s3_secret_access_key or None,
        "region_name": settings.s3_region,
        "verify": not settings.s3_insecure,
    }
    async with session.client("s3", **params) as client:
        kwargs = {"Bucket": bucket, "Key": key}
        if etag:
            kwargs["IfNoneMatch"] = etag
        try:
            resp = await client.get_object(**kwargs)
        except ClientError as exc:
            status = exc.response.get("ResponseMetadata", {}).get("HTTPStatusCode")
            if status == 304:
                return None, etag
            raise FetchError(f"s3 get_object failed: {status}") from exc

        body = await resp["Body"].read()
        text = body.decode(resp.get("ContentEncoding") or "utf-8")
        new_etag = resp.get("ETag")
        if new_etag:
            new_etag = new_etag.strip('"')
        return text, new_etag or etag
