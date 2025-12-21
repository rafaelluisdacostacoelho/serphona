import { describe, expect, it, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';

const loginMock = vi.fn();

vi.mock('../hooks/useAuth', () => ({
  __esModule: true,
  useAuth: () => ({ login: loginMock }),
}));

import { LoginForm } from './LoginForm';

describe('LoginForm', () => {
  it('calls login with email and password', async () => {
    render(<LoginForm />);

    await userEvent.type(screen.getByPlaceholderText(/Email/i), 'demo@serphona.com');
    await userEvent.type(screen.getByPlaceholderText(/Password/i), 'secret123');
    await userEvent.click(screen.getByRole('button', { name: /login/i }));

    expect(loginMock).toHaveBeenCalledWith({ email: 'demo@serphona.com', password: 'secret123' });
  });
});
