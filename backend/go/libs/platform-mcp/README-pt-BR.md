# platform-mcp (esqueleto)

Propósito: blocos compartilhados para MCP (schemas, políticas, discovery, auditoria) para servidores e clientes MCP no Serphona.

Status: somente esqueleto — sem código ainda.

Sugestão de conteúdo (próximos passos):
- `resource/`: tipos para recursos/tools MCP com schemas de entrada/saída e dicas de idempotência.
- `policy/`: helpers para allow/deny por tenant/agente/ambiente, rate limits e escopos.
- `client/`: cliente fino para resolver recursos e executar chamadas com tracing/métricas.
- `audit/`: helpers de logging (hash de input/output, status, latência, IDs de tenant/agente).
- `test/`: contratos/mocks para política e cliente.
