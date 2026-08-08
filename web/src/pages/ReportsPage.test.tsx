import { screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it } from 'vitest';
import { mockFetch, renderWithProviders } from '../test/render';
import ReportsPage from './ReportsPage';

const harbor = {
  property_id: 'p1',
  property_name: 'Harbor View Apartments',
  charged_cents: 900000,
  collected_cents: 850000,
  withheld_cents: 32500,
  expenses_cents: 920000,
  net_cents: -37500,
  by_category: [{ category: 'mortgage', amount_cents: 685000 }],
};

describe('ReportsPage', () => {
  it('shows each property and the totals', async () => {
    const calls = mockFetch({
      'GET /api/reports/profit-loss': {
        body: {
          from: '2026-01-01',
          to: '2026-09-14',
          properties: [harbor],
          totals: { ...harbor, property_name: 'All properties' },
        },
      },
    });
    renderWithProviders(<ReportsPage />, { route: '/reports?period=last-year' });

    const row = (await screen.findByRole('cell', { name: 'Harbor View Apartments' })).closest('tr')!;
    expect(within(row).getByText('$8,500.00')).toBeInTheDocument();
    expect(within(row).getByText('-$375.00')).toHaveClass('negative');
    expect(screen.getByRole('rowheader', { name: 'All properties' })).toBeInTheDocument();
    expect(screen.getByText('Mortgage')).toBeInTheDocument();
    expect(calls[0]?.url).toMatch(/from=\d{4}-01-01&to=\d{4}-12-31/);
  });

  it('asks for a custom period', async () => {
    const calls = mockFetch({
      'GET /api/reports/profit-loss': {
        body: { from: '2026-01-01', to: '2026-01-31', properties: [], totals: { ...harbor, by_category: [] } },
      },
    });
    renderWithProviders(<ReportsPage />);
    await userEvent.selectOptions(await screen.findByLabelText('Period'), 'custom');
    const from = screen.getByLabelText('From');
    await userEvent.clear(from);
    await userEvent.type(from, '2025-07-01');
    expect(calls.at(-1)?.url).toContain('from=2025-07-01');
  });
});
