# Estratégia de Chunking para RAG (rascunho)

Propósito: descrever uma abordagem simples e conservadora de chunking para as fases iniciais de ingestão/indexação; será refinada com corpus real.

Regras básicas:
- Tamanho alvo por chunk: ~1,2k caracteres; overlap: ~80 caracteres para preservar contexto entre limites.
- Normalizar whitespace; remover espaços nas bordas; preferir quebras em `.`, `?`, `!`, quebras de linha ou espaços.
- Recusar TTL negativo; manter `tenant_id`, `namespace`, `document_id`, `version`, `etag`, `tags`, `acl` no metadado; espelhar campos canônicos em attributes para armazenamento.
- Produzir offsets por chunk para futura associação com citações/rerankers.

Próximos passos:
- Ajustar tamanhos por tipo de fonte (FAQ vs. manuais vs. transcrições).
- Adicionar split de frases sensível a idioma e limpeza opcional de HTML/Markdown.
- Integrar no worker Python que consome `rag.ingestion.requested` e faz upsert no pgvector.
