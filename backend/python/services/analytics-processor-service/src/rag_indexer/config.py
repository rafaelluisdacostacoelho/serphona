import os
from dataclasses import dataclass
from typing import List


def _split_csv(value: str) -> List[str]:
    return [v.strip() for v in value.split(",") if v.strip()]


@dataclass
class Settings:
    kafka_bootstrap: str = os.getenv("KAFKA_BOOTSTRAP", "kafka:9092")
    kafka_group_id: str = os.getenv("KAFKA_GROUP_ID", "rag-indexer")
    kafka_topic: str = os.getenv("KAFKA_TOPIC", "rag.ingestion.requested")
    pg_dsn: str = os.getenv("PG_DSN", "postgresql://postgres:postgres@localhost:5432/serphona_rag?sslmode=disable")
    pg_table: str = os.getenv("PG_TABLE", "rag_chunks")
    pg_pool_min: int = int(os.getenv("PG_POOL_MIN", "1"))
    pg_pool_max: int = int(os.getenv("PG_POOL_MAX", "5"))
    chunk_max_chars: int = int(os.getenv("CHUNK_MAX_CHARS", "1200"))
    chunk_overlap: int = int(os.getenv("CHUNK_OVERLAP", "80"))
    chunk_min_boundary: int = int(os.getenv("CHUNK_MIN_BOUNDARY", "120"))
    normalize_whitespace: bool = os.getenv("CHUNK_NORMALIZE_WHITESPACE", "true").lower() in {"1", "true", "yes"}
    trim_space: bool = os.getenv("CHUNK_TRIM_SPACE", "true").lower() in {"1", "true", "yes"}
    fetch_timeout_seconds: int = int(os.getenv("FETCH_TIMEOUT_SECONDS", "10"))
    allowed_schemes: List[str] = None

    s3_endpoint_url: str = os.getenv("S3_ENDPOINT_URL", "")
    s3_region: str = os.getenv("S3_REGION", "us-east-1")
    s3_access_key_id: str = os.getenv("S3_ACCESS_KEY_ID", "")
    s3_secret_access_key: str = os.getenv("S3_SECRET_ACCESS_KEY", "")
    s3_insecure: bool = os.getenv("S3_INSECURE", "false").lower() in {"1", "true", "yes"}

    embedding_provider: str = os.getenv("EMBEDDING_PROVIDER", "noop")
    embedding_model: str = os.getenv("EMBEDDING_MODEL", "text-embedding-3-small")
    embedding_dim: int = int(os.getenv("EMBEDDING_DIM", "1536"))
    embedding_api_key: str = os.getenv("EMBEDDING_API_KEY", "")
    embedding_base_url: str = os.getenv("EMBEDDING_BASE_URL", "")

    etag_cache_path: str = os.getenv("ETAG_CACHE_PATH", "")
    dlq_path: str = os.getenv("DLQ_PATH", "")
    max_retries: int = int(os.getenv("MAX_RETRIES", "3"))
    retry_backoff_seconds: float = float(os.getenv("RETRY_BACKOFF_SECONDS", "1.0"))
    prometheus_port: int = int(os.getenv("PROMETHEUS_PORT", "9000"))

    def __post_init__(self):
        if self.allowed_schemes is None:
            self.allowed_schemes = _split_csv(os.getenv("ALLOWED_URI_SCHEMES", "s3,https"))


settings = Settings()
