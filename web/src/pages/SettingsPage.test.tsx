import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Route, Routes } from 'react-router';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { getToken, setToken } from '../api/client';
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
  time_zone: 'UTC',
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
  vi.restoreAllMocks();
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
    expect(calls.at(-1)?.body).toEqual({
      name: 'Demo Landlord',
      currency: 'EUR',
      deposit_return_days: 30,
      time_zone: 'UTC',
    });
    expect(formatMoney(100)).toBe('€1.00');
  });

  it('saves the time zone', async () => {
    const calls = mockFetch({
      'GET /api/auth/me': { body: user },
      'PUT /api/auth/me': { body: { ...user, time_zone: 'Europe/Bucharest' } },
    });
    renderSettings();

    await userEvent.selectOptions(await screen.findByLabelText('Time zone'), 'Europe/Bucharest');
    await userEvent.click(screen.getByRole('button', { name: 'Save' }));

    expect(await screen.findByText('Saved')).toBeInTheDocument();
    expect(calls.at(-1)?.body).toMatchObject({ time_zone: 'Europe/Bucharest' });
  });

  it("offers the device's time zone when it differs from the saved one", async () => {
    vi.spyOn(Intl.DateTimeFormat.prototype, 'resolvedOptions').mockReturnValue({
      timeZone: 'Pacific/Auckland',
    } as Intl.ResolvedDateTimeFormatOptions);
    mockFetch({ 'GET /api/auth/me': { body: user } });
    renderSettings();

    await userEvent.click(await screen.findByRole('button', { name: 'Use Pacific / Auckland' }));
    expect(screen.getByLabelText('Time zone')).toHaveValue('Pacific/Auckland');
    expect(screen.queryByRole('button', { name: 'Use Pacific / Auckland' })).not.toBeInTheDocument();
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

  it('keeps this session going with the new token', async () => {
    mockFetch({
      'GET /api/auth/me': { body: user },
      'PUT /api/auth/password': { body: { token: 'tok-new', expires_at: '', user } },
    });
    renderSettings();

    await userEvent.type(await screen.findByLabelText('Current password'), 'old-password');
    await userEvent.type(screen.getByLabelText('New password'), 'new-password');
    await userEvent.type(screen.getByLabelText('Repeat new password'), 'new-password');
    await userEvent.click(screen.getByRole('button', { name: 'Change password' }));

    expect(await screen.findByText(/Other devices have been signed out/)).toBeInTheDocument();
    expect(getToken()).toBe('tok-new');
  });
});
