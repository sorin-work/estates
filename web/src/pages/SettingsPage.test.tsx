import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Route, Routes } from 'react-router';
import { afterEach, describe, expect, it } from 'vitest';
import { setToken } from '../api/client';
import { AuthProvider } from '../auth/AuthContext';
import { RequireAuth } from '../auth/guards';
import { formatMoney, setCurrency } from '../lib/money';
import { mockFetch, renderWithProviders } from '../test/render';
import SettingsPage from './SettingsPage';

const user = {
  id: 'u1',
  email: 'demo@example.com',
  name: 'Demo',
  currency: 'USD',
  deposit_return_days: 30,
  created_at: '',
};

function renderSettings() {
  setToken('tok-123');
  return renderWithProviders(
    <AuthProvider>
      <Routes>
        <Route element={<RequireAuth />}>
          <Route path="/settings" element={<SettingsPage />} />
        </Route>
      </Routes>
    </AuthProvider>,
    { route: '/settings' },
  );
}

afterEach(() => {
  setToken(null);
  setCurrency('USD');
});

describe('SettingsPage', () => {
  it('saves the name and currency', async () => {
    const calls = mockFetch({
      'GET /api/auth/me': { body: user },
      'PUT /api/auth/me': { body: { ...user, name: 'Demo Landlord', currency: 'EUR' } },
    });
    renderSettings();

    const name = await screen.findByLabelText('Name');
    await userEvent.clear(name);
    await userEvent.type(name, 'Demo Landlord');
    await userEvent.selectOptions(screen.getByLabelText('Currency'), 'EUR');
    await userEvent.click(screen.getByRole('button', { name: 'Save' }));

    expect(await screen.findByText('Saved')).toBeInTheDocument();
    expect(calls.at(-1)?.body).toEqual({ name: 'Demo Landlord', currency: 'EUR', deposit_return_days: 30 });
    expect(formatMoney(100)).toBe('€1.00');
  });

  it('checks the repeated password before calling the API', async () => {
    const calls = mockFetch({ 'GET /api/auth/me': { body: user } });
    renderSettings();

    await userEvent.type(await screen.findByLabelText('Current password'), 'old-password');
    await userEvent.type(screen.getByLabelText('New password'), 'new-password');
    await userEvent.type(screen.getByLabelText('Repeat new password'), 'new-pasword');
    await userEvent.click(screen.getByRole('button', { name: 'Change password' }));

    expect(await screen.findByText('does not match the new password')).toBeInTheDocument();
    expect(calls.some((c) => c.method === 'PUT')).toBe(false);
  });
});
