import { describe, expect, it } from 'vitest';
import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import Layout from './Layout';

describe('Layout', () => {
  it('renders navigation links without real network', () => {
    render(
      <MemoryRouter>
        <Layout />
      </MemoryRouter>
    );

    expect(screen.getByText('Recursos')).toBeInTheDocument();
    expect(screen.getByText('Preços')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: /Entrar/i })).toHaveAttribute('href', expect.stringContaining('login'));
  });
});
