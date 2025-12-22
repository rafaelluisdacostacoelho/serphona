import { http, HttpResponse } from 'msw';

const API_BASE = import.meta.env.VITE_BILLING_API_URL || 'http://localhost:8081';

export const handlers = [
  http.get(`${API_BASE}/api/v1/plans`, () =>
    HttpResponse.json([
      { id: 'basic', name: 'Basic', price: 10 },
      { id: 'pro', name: 'Pro', price: 50 },
    ])
  ),
];
