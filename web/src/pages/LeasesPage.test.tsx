import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it } from 'vitest';
import { mockFetch, renderWithProviders } from '../test/render';
import LeasesPage from './LeasesPage';

const lease = (id: string, tenant_name: string, property_name: string) => ({
  id,
  unit_id: 'u-' + id,
  unit_label: 'A',
  property_id: 'p-' + property_name,
  property_name,
  tenant_id: 't-' + id,
  tenant_name,
  start_date: '2026-01-01',
  end_date: '2026-12-31',
  rent_cents: 150000,
  current_rent_cents: 150000,
  deposit_cents: 0,
  due_day: 1,
  grace_days: 5,
  late_fee_cents: 0,
  terminated_on: null,
  notes: '',
  status: 'active',
  rent_changes: [],
  renewed_from_id: null,
  renewed_by_id: null,
});

describe('LeasesPage', () => {
  it('filters by property on the server and by name as you type', async () => {
    const calls = mockFetch({
      'GET /api/leases': { body: [lease('1', 'Maria Gonzalez', 'Maple'), lease('2', 'James Whitaker', 'Maple')] },
      'GET /api/properties': { body: [{ id: 'p-Maple', name: 'Maple Street Duplex' }] },
    });
    renderWithProviders(<LeasesPage />);

    expect(await screen.findByText('Maria Gonzalez')).toBeInTheDocument();
    await userEvent.type(screen.getByLabelText('Tenant name'), 'whit');
    expect(screen.queryByText('Maria Gonzalez')).not.toBeInTheDocument();
    expect(screen.getByText('James Whitaker')).toBeInTheDocument();

    await screen.findByRole('option', { name: 'Maple Street Duplex' });
    await userEvent.selectOptions(screen.getByLabelText('Property'), 'p-Maple');
    expect(calls.some((c) => c.url === '/api/leases?property_id=p-Maple')).toBe(true);
  });
});
