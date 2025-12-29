# platform-mcp

Blocos compartilhados para servidores e clientes MCP usados no Serphona.

## O que tem aqui
- errors/: códigos de erro comuns.
- protocol/: schemas compartilhados de requisição/resposta/tool e validação.
- loader/: carregador seguro de manifestos (JSON), respeita MCP_TOOLS_ROOT, bloqueia travessia e valida ferramentas.
- registry/: registries em memória, cacheados, com allow-list e Postgres, todos com suporte a ETag em list/describe.
- invoke/: executores (estático, ciente de cancelamento, observado, resiliente) e helpers de auditoria/progresso.
- policy/: motor de regras em memória com matching por tenant/tool/ambiente, escopos obrigatórios e rate limit por regra.
- response/: envelopes de sucesso/erro com trace/request IDs e paginação.
- session/: stores em memória e Postgres para sessões de invocação.

## Como usar
- Manifestos de ferramentas: forneça JSON; com MCP_TOOLS_ROOT definido, os caminhos ficam confinados à raiz. Arquivos não regulares e travessia são rejeitados.
- Registries: use ListToolsWithETag/DescribeToolWithETag para honrar If-None-Match. CachedRegistry renova por TTL; AllowListRegistry filtra por tenant.
- Invocação: StaticExecutor roteia por nome. Envolva com ObservedExecutor para métricas, CancelableExecutor para cancelamento via contexto e ResilientExecutor para tentativas com backoff/circuit breaker.
- Política: MemoryEvaluator é fail-closed (sem regra -> deny). RateLimitPerMinute aplica janela por tenant+tool+rule e retorna RetryAfter quando limitado.
- Respostas: use Success/Error e aplique WithTraceFromContext/WithRequestID para correlação.
- Sessões: store Postgres usa pgx; testes unitários utilizam mocks, sem DB real.

## Inícios rápidos
- Carregar ferramentas de arquivo:
  - MCP_TOOLS_ROOT=/caminho/para/tools go test ./loader -run LoadFromFile
- Registry com cache:
  - reg := registry.NewCachedRegistry(loader, 30*time.Second)
  - etag, tools, notMod, err := reg.ListToolsWithETag(ctx, "tenant", ifNoneMatch)
- Execução resiliente:
  - exec := invoke.NewResilientExecutor(inner, invoke.ResilientConfig{MaxRetries: 2})
  - ch, err := exec.Invoke(ctx, req)

## Testes
- Rode todos os testes com cobertura:
  - make test-coverage (no diretório deste módulo), ou
  - go test ./... -cover

CI espera que os testes passem sem serviços externos; dependências Postgres/Kafka são mockadas.

Consulte IMPLEMENTATION_GUIDE-pt-BR.md para padrões de integração e pontos de extensão.
