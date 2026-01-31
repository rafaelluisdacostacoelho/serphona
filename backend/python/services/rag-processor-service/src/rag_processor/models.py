"""Data models for RAG ingestion pipeline."""

from dataclasses import dataclass


@dataclass
class DocumentInput:
    tenant_id: str
    namespace: str
    document_id: str
    etag: str
    content: str


@dataclass
class DocumentRecord:
    tenant_id: str
    namespace: str
    document_id: str
    etag: str
    embedding_provider: str
    embedding: list[list[float]]


@dataclass
class ProcessResult:
    status: str  # "skipped" | "updated" | "inserted"
    provider: str
    document_id: str
    etag: str
