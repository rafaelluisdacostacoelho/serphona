# Marketing Website — Backlog (en-US)

## Status snapshot
- Public site (Vite/Tailwind). Not audited in this pass.

## Open items
1) **Performance/SEO**: optimize Lighthouse metrics (LCP/CLS/TTI), metadata/OG tags, sitemap/robots.
2) **Content safety**: sanitize any dynamic content; CSP headers via nginx config; form spam protection.
3) **Analytics/telemetry**: privacy-compliant analytics; feature flags if needed; cookie consent if applicable.
4) **Accessibility**: a11y checks, keyboard navigation, proper semantics.
5) **Internationalization**: ensure locale switching if supported; fallback content.
6) **Testing**: visual regression/e2e for key pages; linting; link checking.
7) **Docs**: env vars, build/deploy steps, CDN/cache guidance.

## Config to surface
- Analytics keys, feature flags, API endpoints (if any forms), CSP headers.

## Test coverage checklist
- [ ] Lighthouse target met
- [ ] a11y smoke tests
- [ ] Analytics initialization
- [ ] SEO (sitemap/OG) checks

## Next steps
- Audit pages/config; populate concrete tasks and mark done/todo; add missing tests and deployment notes.
