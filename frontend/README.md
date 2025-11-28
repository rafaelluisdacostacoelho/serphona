# Serphona Frontend

Este diretório contém dois frontends integrados para o projeto Serphona:

1. **Website** (`frontend/website`) - Site institucional público
2. **Console** (`frontend/console`) - Dashboard de gerenciamento (painel administrativo)

## 🏗️ Arquitetura

### Website (Porta 3000)
- **Propósito**: Site institucional público para apresentar o produto
- **Tecnologias**: React 18, TypeScript, React Router, TailwindCSS, Vite
- **Páginas**:
  - Home (`/`)
  - Recursos (`/features`)
  - Preços (`/pricing`)
  - Sobre (`/about`)
  - Contato (`/contact`)

### Console (Porta 3001)
- **Propósito**: Dashboard para gerenciamento de agentes e configurações
- **Tecnologias**: React 18, TypeScript, React Router, TailwindCSS, Vite, React Query, i18next
- **Funcionalidades**:
  - Autenticação (Login/Register)
  - Dashboard principal
  - Gerenciamento de agentes
  - Configuração de ferramentas/integrações
  - Analytics
  - Faturamento
  - Configurações

## 🚀 Como Executar

### Pré-requisitos
- Node.js 18+ instalado
- npm ou yarn

### 1. Configurar Variáveis de Ambiente

#### Website
```bash
cd frontend/website
cp .env.example .env
```

Edite o arquivo `.env` se necessário para apontar para o console:
```env
VITE_CONSOLE_URL=http://localhost:3001
```

#### Console
```bash
cd frontend/console
cp .env.example .env
```

Edite o arquivo `.env` para apontar para sua API backend:
```env
VITE_API_URL=http://localhost:8080/api/v1
```

### 2. Instalar Dependências

#### Website
```bash
cd frontend/website
npm install
```

#### Console
```bash
cd frontend/console
npm install
```

### 2. Executar em Desenvolvimento

#### Opção A: Executar Ambos Simultaneamente

**Terminal 1 - Website:**
```bash
cd frontend/website
npm run dev
```
O website estará disponível em: `http://localhost:3000`

**Terminal 2 - Console:**
```bash
cd frontend/console
npm run dev
```
O console estará disponível em: `http://localhost:3001`

#### Opção B: Script Helper (Criar este script na raiz do projeto)

Crie um arquivo `start-frontends.sh`:
```bash
#!/bin/bash
cd frontend/website && npm run dev &
cd frontend/console && npm run dev &
wait
```

Execute:
```bash
chmod +x start-frontends.sh
./start-frontends.sh
```

### 3. Build para Produção

#### Website
```bash
cd frontend/website
npm run build
npm run preview  # Para testar o build localmente
```

#### Console
```bash
cd frontend/console
npm run build
npm run preview  # Para testar o build localmente
```

## 🔄 Integração entre Website e Console

A integração entre os dois frontends funciona da seguinte forma:

1. **Do Website para o Console**:
   - Botões "Entrar" e "Começar Grátis" no header do website redirecionam para o console
   - Link "Entrar": `http://localhost:3001/login`
   - Link "Começar Grátis": `http://localhost:3001/register`

2. **Do Console para o Website**:
   - O usuário pode acessar o website institucional através dos links no footer do console

### Configuração de URLs

As URLs estão configuradas nos seguintes arquivos:

**Website** (`frontend/website/src/components/Layout.tsx`):
```typescript
const CONSOLE_URL = 'http://localhost:3001';
```

**Para Produção**, você deve atualizar essas URLs para os domínios reais:
- Website: `https://serphona.com`
- Console: `https://console.serphona.com` ou `https://app.serphona.com`

## 📁 Estrutura de Diretórios

```
frontend/
├── website/                    # Site institucional
│   ├── src/
│   │   ├── components/        # Componentes reutilizáveis
│   │   │   └── Layout.tsx     # Layout principal com header/footer
│   │   ├── pages/             # Páginas do site
│   │   │   ├── HomePage.tsx
│   │   │   ├── FeaturesPage.tsx
│   │   │   ├── PricingPage.tsx
│   │   │   ├── AboutPage.tsx
│   │   │   └── ContactPage.tsx
│   │   ├── App.tsx            # Configuração de rotas
│   │   ├── main.tsx           # Entry point
│   │   └── index.css          # Estilos globais
│   ├── index.html
│   ├── package.json
│   ├── vite.config.ts
│   └── tailwind.config.js
│
└── console/                    # Dashboard administrativo
    ├── src/
    │   ├── components/
    │   │   ├── forms/         # Componentes de formulário
    │   │   └── layout/        # Layout do dashboard
    │   │       └── AppLayout.tsx
    │   ├── context/           # Context API
    │   │   └── AuthContext.tsx
    │   ├── features/          # Features modulares
    │   │   ├── auth/         # Autenticação
    │   │   ├── agents/       # Gerenciamento de agentes
    │   │   ├── analytics/    # Analytics
    │   │   ├── billing/      # Faturamento
    │   │   ├── dashboard/    # Dashboard principal
    │   │   ├── settings/     # Configurações
    │   │   └── tools/        # Ferramentas/Integrações
    │   ├── i18n/             # Internacionalização
    │   ├── routes/           # Configuração de rotas
    │   ├── services/         # Serviços/API
    │   ├── App.tsx
    │   ├── main.tsx
    │   └── index.css
    ├── index.html
    ├── package.json
    ├── vite.config.ts
    └── tailwind.config.js
```

## 🎨 Customização

### Cores do Tema
Ambos os projetos usam a mesma paleta de cores primária (Primary - Indigo):

```css
--primary-50: #eef2ff
--primary-100: #e0e7ff
--primary-200: #c7d2fe
--primary-300: #a5b4fc
--primary-400: #818cf8
--primary-500: #6366f1
--primary-600: #4f46e5 (cor principal)
--primary-700: #4338ca
--primary-800: #3730a3
--primary-900: #312e81
--primary-950: #1e1b4b
```

Para alterar as cores, edite os arquivos:
- `frontend/website/tailwind.config.js`
- `frontend/console/tailwind.config.js`

## 🔐 Autenticação

O fluxo de autenticação funciona da seguinte forma:

1. Usuário acessa o website e clica em "Começar Grátis" ou "Entrar"
2. É redirecionado para o console (`localhost:3001/register` ou `/login`)
3. Após autenticação bem-sucedida, acessa o dashboard completo
4. O AuthContext gerencia o estado de autenticação
5. ProtectedRoute protege rotas que requerem autenticação

## 🌐 Internacionalização (i18n)

O console suporta múltiplos idiomas através do i18next:

**Idiomas disponíveis**:
- Português (pt)
- Inglês (en)

**Arquivos de tradução**:
- `frontend/console/src/i18n/locales/pt.json`
- `frontend/console/src/i18n/locales/en.json`

## 📝 Próximos Passos

### Website
- [ ] Completar página de Recursos com detalhes técnicos
- [ ] Criar página de Preços com planos e comparações
- [ ] Adicionar formulário de contato funcional
- [ ] Implementar seção de FAQ
- [ ] Adicionar depoimentos de clientes

### Console
- [ ] Implementar formulários de criação/edição de agentes
- [ ] Conectar com APIs do backend
- [ ] Implementar visualizações de analytics com gráficos
- [ ] Adicionar gerenciamento de ferramentas/integrações
- [ ] Implementar painel de faturamento
- [ ] Adicionar configurações de perfil e tenant

## 🐛 Solução de Problemas

### Porta já em uso
Se você receber erro de porta já em uso, você pode:

1. Matar o processo na porta:
```bash
# Linux/Mac
lsof -ti:3000 | xargs kill -9
lsof -ti:3001 | xargs kill -9

# Windows
netstat -ano | findstr :3000
taskkill /PID <PID> /F
```

2. Ou alterar a porta no `vite.config.ts`

### Erros de TypeScript
Os erros de TypeScript mostrados no editor são normais antes de executar `npm install`. Após instalar as dependências, eles desaparecerão.

### Problemas com dependências
```bash
# Limpar cache e reinstalar
rm -rf node_modules package-lock.json
npm install
```

## 📚 Documentação Adicional

- [React Documentation](https://react.dev)
- [Vite Documentation](https://vitejs.dev)
- [TailwindCSS Documentation](https://tailwindcss.com)
- [React Router Documentation](https://reactrouter.com)
- [React Query Documentation](https://tanstack.com/query)

## 🤝 Contribuindo

Para contribuir com o frontend:

1. Crie uma branch para sua feature
2. Faça suas alterações
3. Teste em ambos os frontends se aplicável
4. Envie um pull request

## 📄 Licença

Este projeto faz parte do sistema Serphona. Todos os direitos reservados.
