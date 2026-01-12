### Novo script: compose-rebuild.sh (executável).

- Usa docker compose v2; se não existir, faz fallback para docker-compose v1.
- Serviços padrão: tenant-manager, auth-gateway, agent-orchestrator, tools-gateway, billing-service, analytics-processor.
- Você pode passar serviços via args ou SERVICES="a b c".

### Uso:

```sh
./scripts/compose-rebuild.sh            # build/up dos serviços padrão
./scripts/compose-rebuild.sh tools-gateway billing-service
SERVICES="agent-orchestrator tools-gateway" ./scripts/compose-rebuild.sh
```