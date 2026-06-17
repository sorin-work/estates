import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it } from 'vitest';
import { mockFetch, renderWithProviders } from '../test/render';
import VendorsPage from './VendorsPage';

const vendor = {
  id: 'v1',
  name: 'Rapid Rooter Plumbing',
  trade: 'plumbing',
  contact_name: 'Luis Ortega',
  phone: '(503) 555-0110',
  email: '',
  notes: '',
  created_at: '',
  open_jobs: 2,
  spent_cents: 18500,
  last_used_on: '2026-06-28',
};

describe('VendorsPage', () => {
  it('lists vendors with what was spent', async () => {
    mockFetch({ 'GET /api/vendors': { body: [vendor] } });
    renderWithProviders(<VendorsPage />);

    expect(await screen.findByRole('link', { name: 'Rapid Rooter Plumbing' })).toHaveAttribute('href', '/vendors/v1');
    expect(screen.getByRole('cell', { name: 'Plumbing' })).toBeInTheDocument();
    expect(screen.getByText('$185.00')).toBeInTheDocument();
    expect(screen.getByText('Jun 28, 2026')).toBeInTheDocument();
  });

  it('filters by trade', async () => {
    const calls = mockFetch({ 'GET /api/vendors': { body: [] } });
    renderWithProviders(<VendorsPage />);
    expect(await screen.findByText('No vendors yet')).toBeInTheDocument();

    await userEvent.selectOptions(screen.getByLabelText('Trade'), 'hvac');
    expect(await screen.findByText('No vendors in this trade')).toBeInTheDocument();
    expect(calls.at(-1)?.url).toBe('/api/vendors?trade=hvac');
  });
});
