# Serphona Website

Landing/marketing site built with React + Vite.

## Quick Start

```bash
npm install
npm run dev

# Tests (Vitest + RTL + msw)
npm test
npm run test:coverage
```

## Notes
- Tests use jsdom with React Testing Library and msw to avoid real HTTP calls.
- Update `src/test/mocks/handlers.ts` with API stubs as you add covered endpoints.
