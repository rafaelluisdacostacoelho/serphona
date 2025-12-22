# Architecture Documentation

This directory hosts Serphona's architecture specs, design docs, and diagrams. Key topics now include RAG/MCP, observability, telephony, and tooling gateways.

## Contents (highlights)
- Platform overview and diagrams
- RAG + MCP architecture: ingestion, retrieval, MCP tool governance
- Prompts spec (v1.1 with RAG/MCP fields)
- Observability and dashboards
- Telephony extensions (tenant-manager), voice and tools gateways
- Infrastructure/Helm/Terraform notes (see infra/) 

## Documents
- RAG/MCP: [RAG-MCP-en-US.md](RAG-MCP-en-US.md) · [RAG-MCP-pt-BR.md](RAG-MCP-pt-BR.md) · [BACKLOG-RAG-MCP.md](BACKLOG-RAG-MCP.md)
- Prompts spec: [PROMPTS-YAML-SPEC-en-US.md](PROMPTS-YAML-SPEC-en-US.md) · [PROMPTS-YAML-SPEC.md](PROMPTS-YAML-SPEC.md)
- Telephony: [TENANT-MANAGER-TELEPHONY-EXTENSIONS-en-US.md](TENANT-MANAGER-TELEPHONY-EXTENSIONS-en-US.md) · [TENANT-MANAGER-TELEPHONY-EXTENSIONS-pt-BR.md](TENANT-MANAGER-TELEPHONY-EXTENSIONS-pt-BR.md)
- Voice Gateway: [VOICE-GATEWAY-DESIGN-en-US.md](VOICE-GATEWAY-DESIGN-en-US.md) · [VOICE-GATEWAY-DESIGN.md](VOICE-GATEWAY-DESIGN.md)
- Tools Gateway: [TOOLS-GATEWAY-ARCHITECTURE-en-US.md](TOOLS-GATEWAY-ARCHITECTURE-en-US.md) · [TOOLS-GATEWAY-ARCHITECTURE.md](TOOLS-GATEWAY-ARCHITECTURE.md)
- Observability: see platform-observability README and dashboards in backend/go/libs/platform-observability/

## See Also
- [API Documentation](../api/)
- [Architecture Decision Records](../decisions/)
