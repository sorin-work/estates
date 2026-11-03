import { screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it } from 'vitest';
import { mockFetch, renderWithProviders } from '../test/render';
import RentPage from './RentPage';

const balance = (id: string, tenant_name: string, balance_cents: number, overdue_cents: number) => ({
  lease: {
    id,
    tenant_name,
    property_name: 'Harbor View Apartments',
    unit_label: id,
    current_rent_cents: 150000,
  },
  balance_cents,
  overdue_cents,
  oldest_overdue: overdue_cents ? '2026-09-01' : null,
  days_overdue: overdue_cents ? 40 : 0,
});

function tenantsInOrder() {
  const table = screen.getAllByRole('table')[0]!;
  return within(table)
    .getAllByRole('link')
    .map((a) => a.textContent);
}

describe('RentPage', () => {
  it('sorts the rent roll by a column', async () => {
    mockFetch({
      'GET /api/rent/balances': {
        body: [
          balance('102', 'Tom Becker', 279000, 279000),
          balance('101', 'Priya Raman', 0, 0),
          balance('202', 'Sofia Rossi', 69000, 0),
        ],
      },
      'GET /api/payments': { body: { payments: [], total: 0, page_total_cents: 0 } },
    });
    renderWithProviders(<RentPage />);
    await screen.findByText('Tom Becker');
    expect(tenantsInOrder()).toEqual(['Tom Becker', 'Priya Raman', 'Sofia Rossi']);

    await userEvent.click(screen.getByRole('button', { name: 'Tenant' }));
    expect(tenantsInOrder()).toEqual(['Priya Raman', 'Sofia Rossi', 'Tom Becker']);
    expect(screen.getByRole('columnheader', { name: /Tenant/ })).toHaveAttribute('aria-sort', 'ascending');

    await userEvent.click(screen.getByRole('button', { name: 'Balance' }));
    expect(tenantsInOrder()).toEqual(['Tom Becker', 'Sofia Rossi', 'Priya Raman']);
  });
});
