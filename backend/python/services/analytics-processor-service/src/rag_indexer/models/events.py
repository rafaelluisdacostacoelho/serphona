from datetime import datetime
from typing import Dict, List, Optional
from pydantic import BaseModel, Field, HttpUrl


class RAGIngestionRequested(BaseModel):
    tenant_id: str = Field(..., alias="tenant_id")
    namespace: str
    source: Optional[str] = None
    document_id: str
    version: Optional[str] = None
    etag: Optional[str] = None
    uri: Optional[str] = None
    tags: Optional[List[str]] = None
    acl: Optional[List[str]] = None
    ttl_seconds: Optional[int] = Field(default=None, ge=0)
    metadata: Optional[Dict[str, str]] = None
    requested_at: datetime

    class Config:
        allow_population_by_field_name = True
        anystr_strip_whitespace = True
