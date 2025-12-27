### 📁 Estrutura do Diretório `.github/workflows`

O diretório `workflows` armazena os arquivos de configuração dos pipelines CI/CD do GitHub Actions. Cada arquivo é um workflow em formato YAML (`.yml`) e define tarefas automatizadas, como testes, builds ou implantações.

#### 📝 Estrutura Sugerida:
```
.github/
└── workflows/
    ├── build.yml          # Workflow para construção e teste do backend
    ├── deploy.yml         # Workflow para implantação em ambiente de produção
    └── lint.yml           # Workflow para análise de código (ex: TypeScript, ESLint)
```

#### 📄 Exemplo de Arquivo (`build.yml`):
```yaml
name: Node.js CI

on:
  push:
    branches: [ main ]
  pull_request:
    branches: [ main ]

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      - name: Instalar dependências
        run: npm install
      - name: Executar testes
        run: npm test
```

#### 📌 Recomendações:
1. **Nomenclatura Clara**: Use nomes descritivos como `deploy-heroku.yml` ou `unit-tests.yml`.
2. **Modularidade**: Divida workflows complexos em arquivos separados (ex: `lint.yml`, `security.yml`).
3. **Secrets**: Armazene credenciais sensíveis (ex: chaves de API) no GitHub via `Settings > Secrets`.

#### 🔒 Guarda de bump baseada em VERSION (PR → main)
- Workflow: `.github/workflows/pr-bump-label.yml`.
- Dispara em todo `pull_request` para `main` (opened, synchronize, reopened, labeled, unlabeled).
- Job/status: **Check the bump label in the PR**.
- Regras:
  - O PR deve ter um label: `bump-patch`, `bump-minor` ou `bump-major`.
  - Pelo menos um arquivo `backend/go/libs/*/VERSION` deve ser alterado.
  - O delta do VERSION precisa bater com o label (patch = incrementa só patch; minor = incrementa minor; major = incrementa major). Novo VERSION exige `bump-major`.
- Para bloquear merges, adicione esse status check como obrigatório na branch rule/ruleset da `main` (Settings → Branches → Rulesets ou Branch protection → "Require status checks to pass" → selecione **Check the bump label in the PR**).

---

### ✅ Próximos Passos
1. Se o diretório `.github/workflows` não existir, crie-o:
   ```bash
   mkdir -p .github/workflows
   ```
2. Adicione arquivos YAML para seus workflows.
3. Atualize este README conforme necessário.
---