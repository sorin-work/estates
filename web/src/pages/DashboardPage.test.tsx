import { screen, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import type { Dashboard } from '../api/types';
import { mockFetch, renderWithProviders } from '../test/render';
import DashboardPage from './DashboardPage';

const months = Array.from({ length: 12 }, (_, i) => ({
  month: `2026-${String(i + 1).padStart(2, '0')}`,
  charged_cents: 1500000,
  collected_cents: 1400000,
  expenses_cents: 300000,
}));

const dashboard: Dashboard = {
  as_of: '2026-12-10',
  occupancy: { units: 10, occupied: 9, vacant: 1, rate: 0.9 },
  this_month: { month: '2026-12', charged_cents: 1500000, collected_cents: 1200000, expenses_cents: 250000 },
  outstanding: { balance_cents: 420000, overdue_cents: 279000, overdue_leases: 1 },
  deposits: { held_cents: 1475000, leases: 9 },
  overdue: [
    {
      lease: {
        id: 'l1',
        unit_id: 'u1',
        unit_label: '102',
        property_id: 'p1',
        property_name: 'Harbor View Apartments',
        tenant_id: 't1',
        tenant_name: 'Tom Becker',
        start_date: '2026-04-01',
        end_date: '2027-03-31',
        rent_cents: 139500,
        deposit_cents: 100000,
        due_day: 1,
        grace_days: 3,
        late_fee_cents: 5000,
        terminated_on: null,
        notes: '',
        status: 'active',
        current_rent_cents: 139500,
        rent_changes: [],
        renewed_from_id: null,
        renewed_by_id: null,
      },
      balance_cents: 279000,
      overdue_cents: 279000,
      oldest_overdue: '2026-11-01',
      days_overdue: 39,
    },
  ],
  expiring_leases: [],
  maintenance: {
    open: 3,
    by_priority: { urgent: 1, high: 1, normal: 1, low: 0 },
    recent: [],
  },
  monthly: months,
};

describe('DashboardPage', () => {
  it('shows occupancy, collections and who is behind', async () => {
    mockFetch({ 'GET /api/dashboard': { body: dashboard } });
    renderWithProviders(<DashboardPage />);

    expect(await screen.findByText('90%')).toBeInTheDocument();
    expect(screen.getByText('9 of 10 units let · 1 vacant')).toBeInTheDocument();
    expect(screen.getByText('$12,000.00')).toBeInTheDocument();
    expect(screen.getByText('of $15,000.00 charged (80%)')).toBeInTheDocument();
    expect(screen.getByText('$14,750.00')).toBeInTheDocument();
    expect(screen.getByText('for 9 leases, not income')).toBeInTheDocument();

    const overdue = screen.getByRole('heading', { name: 'Overdue rent' }).closest('section')!;
    expect(within(overdue).getByText('Tom Becker')).toBeInTheDocument();
    expect(within(overdue).getByText('39 days')).toBeInTheDocument();
    expect(screen.getByText('No leases end in the next 60 days.')).toBeInTheDocument();
  });

  it('welcomes new users with no units', async () => {
    mockFetch({
      'GET /api/dashboard': { body: { ...dashboard, occupancy: { units: 0, occupied: 0, vacant: 0, rate: 0 } } },
    });
    renderWithProviders(<DashboardPage />);
    expect(await screen.findByText('Welcome to Estates')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'adding a property' })).toHaveAttribute('href', '/properties');
  });
});
