from pydantic import BaseModel, Field
from typing import Optional, Dict, Any
from datetime import datetime
from uuid import UUID


class BaseEvent(BaseModel):
    event_id: UUID
    tenant_id: str
    timestamp: datetime
    event_type: str
    metadata: Optional[Dict[str, Any]] = None


class CallEvent(BaseEvent):
    event_type: str = Field("call", const=True)
    call_id: str
    agent_id: Optional[str]
    customer_id: Optional[str]
    transcript: Optional[str]
    duration_ms: Optional[int]
    call_quality: Optional[Dict[str, Any]]


class AgentActionEvent(BaseEvent):
    event_type: str = Field("agent_action", const=True)
    action: str
    agent_id: str
    target_id: Optional[str]


class ToolInvocationEvent(BaseEvent):
    event_type: str = Field("tool_invocation", const=True)
    tool_name: str
    duration_ms: Optional[int]
    success: Optional[bool]


class QoSMetricEvent(BaseEvent):
    event_type: str = Field("qos_metric", const=True)
    metric_name: str
    value: float
    tags: Optional[Dict[str, str]]
