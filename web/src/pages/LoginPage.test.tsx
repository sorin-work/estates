import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Route, Routes } from 'react-router';
import { describe, expect, it } from 'vitest';
import { getToken } from '../api/client';
import { AuthProvider } from '../auth/AuthContext';
import { mockFetch, renderWithProviders } from '../test/render';
import LoginPage from './LoginPage';

function renderLogin() {
  return renderWithProviders(
    <AuthProvider>
      <Routes>
        <Route path="/login" element={<LoginPage />} />
        <Route path="/" element={<div>Dashboard home</div>} />
      </Routes>
    </AuthProvider>,
    { route: '/login' },
  );
}

describe('LoginPage', () => {
  it('signs in and goes to the dashboard', async () => {
    const calls = mockFetch({
      'POST /api/auth/login': {
        body: {
          token: 'tok-123',
          expires_at: '2026-12-01T00:00:00Z',
          user: { id: 'u1', email: 'demo@example.com', name: 'Demo', created_at: '' },
        },
      },
    });
    renderLogin();

    await userEvent.type(screen.getByLabelText('Email'), 'demo@example.com');
    await userEvent.type(screen.getByLabelText('Password'), 'demo-password');
    await userEvent.click(screen.getByRole('button', { name: 'Sign in' }));

    expect(await screen.findByText('Dashboard home')).toBeInTheDocument();
    expect(calls[0]?.body).toEqual({ email: 'demo@example.com', password: 'demo-password' });
    expect(getToken()).toBe('tok-123');
  });

  it('shows the error for wrong credentials', async () => {
    mockFetch({
      'POST /api/auth/login': {
        status: 401,
        body: { error: { code: 'invalid_credentials', message: 'email or password is incorrect' } },
      },
    });
    renderLogin();

    await userEvent.type(screen.getByLabelText('Email'), 'demo@example.com');
    await userEvent.type(screen.getByLabelText('Password'), 'wrong');
    await userEvent.click(screen.getByRole('button', { name: 'Sign in' }));

    expect(await screen.findByRole('alert')).toHaveTextContent('email or password is incorrect');
    expect(screen.queryByText('Dashboard home')).not.toBeInTheDocument();
  });
});
