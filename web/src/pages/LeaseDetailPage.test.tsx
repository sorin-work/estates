import { screen, within } from '@testing-library/react';
import { Route, Routes } from 'react-router';
import { describe, expect, it } from 'vitest';
import { mockFetch, renderWithProviders } from '../test/render';
import LeaseDetailPage from './LeaseDetailPage';

const lease = {
  id: 'l1',
  unit_id: 'u1',
  unit_label: 'A',
  property_id: 'p1',
  property_name: 'Maple Street Duplex',
  tenant_id: 't1',
  tenant_name: 'Maria Gonzalez',
  start_date: '2025-03-01',
  end_date: '2027-02-28',
  rent_cents: 180000,
  current_rent_cents: 185400,
  deposit_cents: 180000,
  due_day: 1,
  grace_days: 5,
  late_fee_cents: 7500,
  terminated_on: null,
  notes: '',
  status: 'active',
  rent_changes: [{ id: 'rc1', effective_date: '2026-03-01', rent_cents: 185400, notes: 'Annual increase' }],
  renewed_from_id: null,
  renewed_by_id: null,
};

function renderLease() {
  mockFetch({
    'GET /api/leases/l1': { body: lease },
    'GET /api/leases/l1/ledger': {
      body: {
        as_of: '2026-11-08',
        charged_cents: 185400,
        paid_cents: 185400,
        balance_cents: 0,
        overdue_cents: 0,
        oldest_overdue: null,
        days_overdue: 0,
        charges: [],
        payments: [],
      },
    },
    'GET /api/leases/l1/deposit': {
      body: {
        lease_id: 'l1',
        agreed_cents: 180000,
        received_cents: 180000,
        returned_cents: 0,
        withheld_cents: 0,
        held_cents: 180000,
        transactions: [
          {
            id: 'd1',
            lease_id: 'l1',
            kind: 'received',
            amount_cents: 180000,
            occurred_on: '2025-02-26',
            notes: 'Paid at signing',
            created_at: '',
          },
        ],
      },
    },
    'GET /api/activity': { body: [] },
    'GET /api/documents': { body: [] },
  });
  renderWithProviders(
    <Routes>
      <Route path="/leases/:id" element={<LeaseDetailPage />} />
    </Routes>,
    { route: '/leases/l1' },
  );
}

describe('LeaseDetailPage', () => {
  it('shows the rent in force, its changes and the deposit held', async () => {
    renderLease();
    expect(await screen.findByRole('heading', { name: /Maria Gonzalez/ })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Renew' })).toBeInTheDocument();

    const changes = screen.getByRole('heading', { name: 'Rent changes' }).closest('section')!;
    expect(within(changes).getByText('Annual increase')).toBeInTheDocument();
    expect(within(changes).getByText('+3.0%')).toBeInTheDocument();

    const deposit = (await screen.findByRole('heading', { name: 'Deposit' })).closest('section')!;
    expect(await within(deposit).findByText('Paid at signing')).toBeInTheDocument();
    expect(within(deposit).getByRole('button', { name: 'Return or withhold' })).toBeInTheDocument();
    expect(screen.getAllByText('$1,800.00').length).toBeGreaterThan(0);
  });
});
