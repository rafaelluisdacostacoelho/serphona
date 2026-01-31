"""Configuration for rag-processor-service."""

from pydantic import Field, HttpUrl
from pydantic_settings import BaseSettings, SettingsConfigDict


class EmbeddingProviderSettings(BaseSettings):
    """Embedding provider configuration with basic quota knobs."""

    model_config = SettingsConfigDict(env_nested_delimiter="__", extra="ignore")

    name: str = Field("openai", description="Logical provider name")
    model: str = Field("text-embedding-3-small", description="Embedding model identifier")
    endpoint: HttpUrl | str | None = Field(None, description="Override endpoint if using self-hosted/OSS provider")
    api_key: str | None = Field(None, description="API key or token")
    timeout_seconds: float = Field(15.0, ge=0.1, description="Request timeout in seconds")
    max_parallel_requests: int = Field(4, ge=1, description="Concurrency limit when batching texts")
    max_tokens_per_minute: int | None = Field(None, ge=1, description="Quota guardrail; None disables the limiter")
    cache_enabled: bool = Field(True, description="Allow cached embeddings when available")
    enabled: bool = Field(True, description="Disable to force fallback provider")


class CacheSettings(BaseSettings):
    model_config = SettingsConfigDict(env_nested_delimiter="__", extra="ignore")

    enabled: bool = Field(True, description="Enable embedding cache")
    backend: str = Field("memory", description="Cache backend: memory|redis|noop")
    redis_url: str | None = Field(None, description="Redis connection URL when backend=redis")
    redis_prefix: str = Field("rag:embed", description="Redis key prefix")
    ttl_seconds: int | None = Field(86400, ge=1, description="TTL for cached embeddings; None disables TTL")


class DlqSettings(BaseSettings):
    model_config = SettingsConfigDict(env_nested_delimiter="__", extra="ignore")

    backend: str = Field("noop", description="DLQ backend: noop|kafka|sqs")
    kafka_topic: str = Field("rag.processor.dlq", description="Kafka topic for DLQ events")
    kafka_bootstrap_servers: str | None = Field(None, description="Kafka bootstrap servers if using kafka backend")
    sqs_queue_url: str | None = Field(None, description="SQS queue URL if using sqs backend")


class AppSettings(BaseSettings):
    """Top-level settings for the worker."""

    model_config = SettingsConfigDict(env_prefix="RAG_", env_nested_delimiter="__", extra="ignore")

    log_level: str = Field("INFO", description="Logging level")
    embedding_primary: EmbeddingProviderSettings = Field(default_factory=EmbeddingProviderSettings)
    embedding_fallback: EmbeddingProviderSettings | None = Field(
        None, description="Optional fallback provider when primary is disabled or fails"
    )
    embedding_cache: CacheSettings = Field(default_factory=CacheSettings)
    dlq: DlqSettings = Field(default_factory=DlqSettings)


def load_settings() -> AppSettings:
    """Load settings from environment variables."""

    return AppSettings()
