from datetime import datetime
from typing import Dict, List, Optional
from pydantic import BaseModel, Field, HttpUrl, ConfigDict


class RAGIngestionRequested(BaseModel):
    model_config = ConfigDict(populate_by_name=True, str_strip_whitespace=True)

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

