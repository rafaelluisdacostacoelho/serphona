import logging
import time
from collections import defaultdict, deque
from typing import Deque, Dict, List, Tuple

import httpx

from .config import settings
from . import metrics

log = logging.getLogger(__name__)


class EmbeddingError(Exception):
    pass


class _QuotaState:
    def __init__(self):
        self.tokens_today = 0
        self.day_ordinal = _today_ordinal()
        self.calls_window: Deque[float] = deque()


_quota_by_tenant: Dict[Tuple[str, str], _QuotaState] = defaultdict(_QuotaState)


def reset_quotas():  # pragma: no cover - used in tests
    _quota_by_tenant.clear()


def embed_texts(tenant_id: str, namespace: str, texts: List[str]) -> List[List[float]]:
    tokens = _estimate_tokens(texts)
    _enforce_daily_quota(tenant_id, namespace, tokens)
    _enforce_tps(tenant_id, namespace)

    provider = settings.embedding_provider.lower().strip()
    last_err = None
    for name, fn in _provider_chain(provider):
        try:
            embeddings = fn(texts)
            _record_cost(tenant_id, namespace, tokens)
            return embeddings
        except EmbeddingError as exc:
            last_err = exc
            log.warning("embedding provider %s failed: %s", name, exc)
            continue

    raise last_err or EmbeddingError("no embedding provider available")


def _provider_chain(provider: str):
    chain = []
    if provider in {"openai", "auto", ""}:
        chain.append(("openai", _embed_openai))
        chain.append(("azure", _embed_azure))
        chain.append(("oss", _embed_oss))
    elif provider == "azure":
        chain.append(("azure", _embed_azure))
        chain.append(("oss", _embed_oss))
    elif provider == "oss":
        chain.append(("oss", _embed_oss))
    else:
        chain.append((provider, _unsupported(provider)))
    return chain


def _unsupported(provider: str):
    def _fn(_texts: List[str]) -> List[List[float]]:
        raise EmbeddingError(f"unsupported embedding provider: {provider}")

    return _fn


def _estimate_tokens(texts: List[str]) -> int:
    # Approximate tokens by characters/4 to avoid external deps.
    chars = sum(len(t or "") for t in texts)
    return max(1, int(chars / 4))


def _enforce_daily_quota(tenant_id: str, namespace: str, tokens: int):
    state = _quota_by_tenant[(tenant_id, namespace)]
    today = _today_ordinal()
    if state.day_ordinal != today:
        state.day_ordinal = today
        state.tokens_today = 0
    if state.tokens_today + tokens > settings.embedding_tokens_per_day:
        metrics.EMBED_QUOTA_EXCEEDED.labels(tenant_id=tenant_id, namespace=namespace, kind="daily_tokens").inc()
        raise EmbeddingError("embedding daily token quota exceeded")
    state.tokens_today += tokens
    metrics.EMBED_TOKENS.labels(tenant_id=tenant_id, namespace=namespace).inc(tokens)


def _enforce_tps(tenant_id: str, namespace: str):
    state = _quota_by_tenant[(tenant_id, namespace)]
    now = time.time()
    window = state.calls_window
    window.append(now)
    # prune entries older than 1 second
    one_sec = now - 1.0
    while window and window[0] < one_sec:
        window.popleft()
    if len(window) > settings.embedding_tps_per_tenant:
        metrics.EMBED_QUOTA_EXCEEDED.labels(tenant_id=tenant_id, namespace=namespace, kind="tps").inc()
        window.pop()  # revert the enqueue for fairness
        raise EmbeddingError("embedding TPS quota exceeded")


def _record_cost(tenant_id: str, namespace: str, tokens: int):
    cost = (tokens / 1000.0) * settings.embedding_cost_per_1k_usd
    metrics.EMBED_COST_USD.labels(tenant_id=tenant_id, namespace=namespace).inc(cost)


def _embed_openai(texts: List[str]) -> List[List[float]]:
    url = settings.embedding_base_url or "https://api.openai.com/v1/embeddings"
    headers = {
        "Authorization": f"Bearer {settings.embedding_api_key}",
        "Content-Type": "application/json",
    }
    payload = {"model": settings.embedding_model, "input": texts}
    try:
        resp = httpx.post(url, headers=headers, json=payload, timeout=30)
        resp.raise_for_status()
        data = resp.json()
        embeddings = [item["embedding"] for item in data.get("data", [])]
        if len(embeddings) != len(texts):
            raise EmbeddingError("embedding count mismatch")
        return embeddings
    except httpx.HTTPError as exc:
        raise EmbeddingError(f"embedding request failed: {exc}") from exc
    except Exception as exc:
        raise EmbeddingError(f"embedding parse failed: {exc}") from exc


def _embed_azure(texts: List[str]) -> List[List[float]]:
    if not settings.embedding_base_url or not settings.embedding_api_key:
        raise EmbeddingError("azure embedding not configured")
    headers = {
        "api-key": settings.embedding_api_key,
        "Content-Type": "application/json",
    }
    payload = {"model": settings.embedding_model, "input": texts}
    url = settings.embedding_base_url
    if "openai" not in url:
        url = url.rstrip("/") + "/embeddings"
    try:
        resp = httpx.post(url, headers=headers, json=payload, timeout=30)
        resp.raise_for_status()
        data = resp.json()
        embeddings = [item.get("embedding") or item.get("data", {}).get("embedding") for item in data.get("data", [])]
        if len(embeddings) != len(texts):
            raise EmbeddingError("embedding count mismatch")
        return embeddings
    except httpx.HTTPError as exc:
        raise EmbeddingError(f"azure embedding request failed: {exc}") from exc
    except Exception as exc:
        raise EmbeddingError(f"azure embedding parse failed: {exc}") from exc


def _embed_oss(texts: List[str]) -> List[List[float]]:
    dim = settings.embedding_dim
    return [[0.0] * dim for _ in texts]


def _today_ordinal() -> int:
    return int(time.time() // 86400)
