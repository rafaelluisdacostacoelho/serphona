import asyncio
import logging
from typing import Optional, Tuple
from urllib.parse import urlparse

import aiohttp
import aioboto3
from botocore.exceptions import ClientError

from .config import settings

log = logging.getLogger(__name__)


class FetchError(Exception):
    pass


async def fetch_content(uri: str, etag: Optional[str] = None) -> Tuple[Optional[str], Optional[str]]:
    """Fetch content from an allowed URI.

    Returns (text, new_etag). If server replies 304/NotModified, returns (None, etag).
    Raises FetchError on disallowed scheme or network errors.
    """

    parsed = urlparse(uri)
    if parsed.scheme not in settings.allowed_schemes:
        raise FetchError(f"scheme not allowed: {parsed.scheme}")

    if parsed.scheme in {"http", "https"}:
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
