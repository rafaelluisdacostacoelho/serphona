import pytest

from rag_processor.config import AppSettings, EmbeddingProviderSettings, load_settings
from rag_processor.embedding import EmbeddingProviderChain


@pytest.fixture(autouse=True)
def clean_env(monkeypatch):
    keys = [
        "RAG_EMBEDDING_PRIMARY__NAME",
        "RAG_EMBEDDING_PRIMARY__MODEL",
        "RAG_EMBEDDING_PRIMARY__ENABLED",
        "RAG_EMBEDDING_FALLBACK__NAME",
        "RAG_EMBEDDING_FALLBACK__ENABLED",
    ]
    for key in keys:
        monkeypatch.delenv(key, raising=False)


def test_load_settings_defaults():
    settings = load_settings()
    assert settings.embedding_primary.name
    assert settings.embedding_primary.model


def test_env_overrides_primary_model(monkeypatch):
    monkeypatch.setenv("RAG_EMBEDDING_PRIMARY__MODEL", "test-model")
    settings = load_settings()
    assert settings.embedding_primary.model == "test-model"


def test_fallback_used_when_primary_disabled(monkeypatch):
    monkeypatch.setenv("RAG_EMBEDDING_PRIMARY__ENABLED", "false")
    monkeypatch.setenv("RAG_EMBEDDING_FALLBACK__NAME", "azure-openai")
    fallback = EmbeddingProviderSettings(name="azure-openai", enabled=True)
    settings = AppSettings(embedding_primary=EmbeddingProviderSettings(enabled=False), embedding_fallback=fallback)
    chain = EmbeddingProviderChain(settings.embedding_primary, settings.embedding_fallback)
    result = chain.embed(["hello"])
    assert result.provider == "azure-openai"


def test_no_active_provider_raises(monkeypatch):
    settings = AppSettings(embedding_primary=EmbeddingProviderSettings(enabled=False), embedding_fallback=None)
    chain = EmbeddingProviderChain(settings.embedding_primary, settings.embedding_fallback)
    with pytest.raises(RuntimeError):
        chain.embed(["hello"])
