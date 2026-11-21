import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Route, Routes } from 'react-router';
import { afterEach, describe, expect, it } from 'vitest';
import { getToken, setToken } from '../api/client';
import { AuthProvider } from '../auth/AuthContext';
import { mockFetch, renderWithProviders } from '../test/render';
import ForgotPasswordPage from './ForgotPasswordPage';
import ResetPasswordPage from './ResetPasswordPage';

const user = {
  id: 'u1',
  email: 'owner@example.com',
  name: 'Owner',
  currency: 'USD',
  deposit_return_days: 30,
  time_zone: 'UTC',
  created_at: '',
};

function renderAt(route: string) {
  return renderWithProviders(
    <AuthProvider>
      <Routes>
        <Route path="/forgot-password" element={<ForgotPasswordPage />} />
        <Route path="/reset-password" element={<ResetPasswordPage />} />
        <Route path="/" element={<div>Dashboard home</div>} />
      </Routes>
    </AuthProvider>,
    { route },
  );
}

afterEach(() => setToken(null));

describe('ForgotPasswordPage', () => {
  it('says a link is on its way', async () => {
    const calls = mockFetch({
      'POST /api/auth/forgot': {
        status: 202,
        body: { message: 'If an account exists for that email, a link is on its way.' },
      },
    });
    renderAt('/forgot-password');

    await userEvent.type(screen.getByLabelText('Email'), 'owner@example.com');
    await userEvent.click(screen.getByRole('button', { name: 'Send the link' }));

    expect(await screen.findByRole('status')).toHaveTextContent('a link is on its way');
    expect(calls[0]?.body).toEqual({ email: 'owner@example.com' });
  });
});

describe('ResetPasswordPage', () => {
  it('sets the new password with the token from the link and signs in', async () => {
    const calls = mockFetch({
      'POST /api/auth/reset': { body: { token: 'tok-new', expires_at: '', user } },
    });
    renderAt('/reset-password#token=abc123');

    await userEvent.type(screen.getByLabelText('New password'), 'new-password');
    await userEvent.type(screen.getByLabelText('Repeat new password'), 'new-password');
    await userEvent.click(screen.getByRole('button', { name: 'Set the new password' }));

    expect(await screen.findByText('Dashboard home')).toBeInTheDocument();
    expect(calls.find((c) => c.method === 'POST')?.body).toEqual({ token: 'abc123', new_password: 'new-password' });
    expect(getToken()).toBe('tok-new');
  });

  it('checks the repeated password first', async () => {
    const calls = mockFetch({});
    renderAt('/reset-password#token=abc123');

    await userEvent.type(screen.getByLabelText('New password'), 'new-password');
    await userEvent.type(screen.getByLabelText('Repeat new password'), 'other-password');
    await userEvent.click(screen.getByRole('button', { name: 'Set the new password' }));

    expect(await screen.findByText('does not match the new password')).toBeInTheDocument();
    expect(calls.some((c) => c.method === 'POST')).toBe(false);
  });

  it('shows the error for a used or expired link', async () => {
    mockFetch({
      'POST /api/auth/reset': {
        status: 400,
        body: { error: { code: 'invalid_reset', message: 'this link is invalid or has expired, ask for a new one' } },
      },
    });
    renderAt('/reset-password#token=abc123');

    await userEvent.type(screen.getByLabelText('New password'), 'new-password');
    await userEvent.type(screen.getByLabelText('Repeat new password'), 'new-password');
    await userEvent.click(screen.getByRole('button', { name: 'Set the new password' }));

    expect(await screen.findByRole('alert')).toHaveTextContent('invalid or has expired');
  });

  it('says so when the link has no token', () => {
    mockFetch({});
    renderAt('/reset-password');
    expect(screen.getByRole('alert')).toHaveTextContent('This link is not complete');
    expect(screen.queryByLabelText('New password')).not.toBeInTheDocument();
  });
});
