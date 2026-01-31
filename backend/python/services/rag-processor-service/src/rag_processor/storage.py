"""Simple document storage abstraction with in-memory implementation."""

from __future__ import annotations

from typing import Dict, Tuple

from .models import DocumentRecord


class DocumentStore:
    """Abstract store for document records."""

    def get(self, tenant_id: str, namespace: str, document_id: str) -> DocumentRecord | None:
        raise NotImplementedError

    def upsert(self, record: DocumentRecord) -> None:
        raise NotImplementedError


class InMemoryDocumentStore(DocumentStore):
    """In-memory store keyed by (tenant, namespace, document_id)."""

    def __init__(self) -> None:
        self._data: Dict[Tuple[str, str, str], DocumentRecord] = {}

    def get(self, tenant_id: str, namespace: str, document_id: str) -> DocumentRecord | None:
        return self._data.get((tenant_id, namespace, document_id))

    def upsert(self, record: DocumentRecord) -> None:
        self._data[(record.tenant_id, record.namespace, record.document_id)] = record
