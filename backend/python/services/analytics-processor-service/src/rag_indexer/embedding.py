import logging
from typing import List

import httpx

from .config import settings

log = logging.getLogger(__name__)


class EmbeddingError(Exception):
    pass


def embed_texts(texts: List[str]) -> List[List[float]]:
    if settings.embedding_provider == "noop" or not settings.embedding_api_key:
        return [[0.0] * settings.embedding_dim for _ in texts]

    if settings.embedding_provider == "openai":
        return _embed_openai(texts)

    raise EmbeddingError(f"unsupported embedding provider: {settings.embedding_provider}")


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
