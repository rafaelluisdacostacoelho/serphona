"""Embedding provider selection and stubs."""

from __future__ import annotations

from dataclasses import dataclass
from typing import Iterable, Sequence

from ..config import EmbeddingProviderSettings


@dataclass
class EmbeddingResult:
    provider: str
    vectors: list[list[float]]


class EmbeddingProviderChain:
    """Selects primary or fallback embedding provider with basic enablement checks."""

    def __init__(
        self,
        primary: EmbeddingProviderSettings,
        fallback: EmbeddingProviderSettings | None = None,
    ) -> None:
        self.primary = primary
        self.fallback = fallback

    def _active_provider(self) -> EmbeddingProviderSettings:
        if self.primary.enabled:
            return self.primary
        if self.fallback and self.fallback.enabled:
            return self.fallback
        raise RuntimeError("no active embedding provider configured")

    def embed(self, texts: Sequence[str]) -> EmbeddingResult:
        """Stub embed call that only returns shape info; replace with real client."""

        provider = self._active_provider()
        cleaned: list[str] = [t for t in texts if t.strip()]
        # Return zero vectors as placeholder; real implementation should call provider endpoint.
        vectors = [[0.0] for _ in cleaned]
        return EmbeddingResult(provider=provider.name, vectors=vectors)

    def iter_providers(self) -> Iterable[EmbeddingProviderSettings]:
        if self.primary.enabled:
            yield self.primary
        if self.fallback and self.fallback.enabled:
            yield self.fallback
