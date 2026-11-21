import { screen, within } from '@testing-library/react';
import { Route, Routes } from 'react-router';
import { describe, expect, it } from 'vitest';
import { mockFetch, renderWithProviders } from '../test/render';
import StatementPage from './StatementPage';

const statement = {
  from: '2026-01-01',
  to: '2026-02-28',
  issued_on: '2026-03-10',
  opening_cents: 0,
  charged_cents: 200000,
  paid_cents: 150000,
  closing_cents: 50000,
  lines: [
    {
      id: 'c1',
      date: '2026-01-01',
      payment: false,
      kind: 'rent',
      description: '',
      amount_cents: 100000,
      balance_cents: 100000,
    },
    {
      id: 'p1',
      date: '2026-01-01',
      payment: true,
      kind: 'bank_transfer',
      description: '',
      amount_cents: 100000,
      balance_cents: 0,
    },
    {
      id: 'c2',
      date: '2026-02-01',
      payment: false,
      kind: 'rent',
      description: '',
      amount_cents: 100000,
      balance_cents: 100000,
    },
    {
      id: 'p2',
      date: '2026-02-04',
      payment: true,
      kind: 'check',
      description: '#104',
      amount_cents: 50000,
      balance_cents: 50000,
    },
  ],
  lease: { id: 'l1', start_date: '2026-01-01', end_date: '2026-12-31' },
  landlord: { name: 'Demo Landlord', email: 'demo@example.com' },
  tenant: { name: 'Grace Hopper', email: 'grace@example.com' },
  property: {
    name: 'Cedar Flats',
    address_line1: '8 Cedar Rd',
    address_line2: '',
    city: 'Eugene',
    region: 'OR',
    postal_code: '97401',
    unit_label: '2',
  },
};

function renderStatement(route = '/leases/l1/statement') {
  return renderWithProviders(
    <Routes>
      <Route path="/leases/:id/statement" element={<StatementPage />} />
    </Routes>,
    { route },
  );
}

describe('StatementPage', () => {
  it('lists charges and payments with the running balance', async () => {
    mockFetch({ 'GET /api/leases/l1/statement': { body: statement } });
    renderStatement();

    expect(await screen.findByRole('heading', { name: 'Rent statement' })).toBeInTheDocument();
    expect(screen.getByText('Grace Hopper')).toBeInTheDocument();
    const rows = within(screen.getByRole('table')).getAllByRole('row');
    // Header, opening balance, four lines, totals.
    expect(rows).toHaveLength(7);
    expect(rows[5]).toHaveTextContent('Payment, check (#104)');
    expect(rows[5]).toHaveTextContent('$500.00');
    expect(screen.getByText('$500.00 owed')).toBeInTheDocument();
  });

  it('asks for the period picked', async () => {
    const calls = mockFetch({ 'GET /api/leases/l1/statement': { body: statement } });
    renderStatement('/leases/l1/statement?from=2026-02-01&to=2026-02-28');

    await screen.findByRole('heading', { name: 'Rent statement' });
    expect(calls[0]?.url).toContain('from=2026-02-01');
    expect(calls[0]?.url).toContain('to=2026-02-28');
    expect(screen.getByLabelText('From')).toHaveValue('2026-02-01');
  });

  it('says when the tenant is in credit', async () => {
    mockFetch({
      'GET /api/leases/l1/statement': { body: { ...statement, closing_cents: -25000, lines: [] } },
    });
    renderStatement();
    expect(await screen.findByText('$250.00 in credit')).toBeInTheDocument();
    expect(screen.getByText('No charges or payments in this period.')).toBeInTheDocument();
  });
});
