import { http, HttpResponse } from 'msw';

export const handlers = [
  http.post('http://localhost:8080/auth/login', async ({ request }) => {
    const body = await request.json();
    return HttpResponse.json({
      user: { id: 'user-1', email: body.email },
      tokens: { accessToken: 'token', refreshToken: 'refresh' },
    });
  }),
];
