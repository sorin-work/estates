import { screen } from '@testing-library/react';
import { Route, Routes } from 'react-router';
import { describe, expect, it } from 'vitest';
import { AuthProvider } from '../auth/AuthContext';
import { mockFetch, renderWithProviders } from '../test/render';
import RegisterPage from './RegisterPage';

function renderRegister() {
  return renderWithProviders(
    <AuthProvider>
      <Routes>
        <Route path="/register" element={<RegisterPage />} />
      </Routes>
    </AuthProvider>,
    { route: '/register' },
  );
}

describe('RegisterPage', () => {
  it('shows the form while sign-up is open', async () => {
    mockFetch({ 'GET /api/auth/config': { body: { registration_open: true } } });
    renderRegister();
    expect(await screen.findByRole('heading', { name: 'Create your account' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Create account' })).toBeInTheDocument();
  });

  it('says sign-up is closed instead of showing the form', async () => {
    mockFetch({ 'GET /api/auth/config': { body: { registration_open: false } } });
    renderRegister();
    expect(await screen.findByRole('heading', { name: 'Sign-up is closed' })).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Create account' })).not.toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Sign in' })).toHaveAttribute('href', '/login');
  });
});
