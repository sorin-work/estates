import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it } from 'vitest';
import { mockFetch, renderWithProviders } from '../test/render';
import RecurringExpenses from './RecurringExpenses';

const mortgage = {
  id: 'r1',
  property_id: 'p1',
  property_name: 'Harbor View Apartments',
  vendor_id: null,
  vendor: 'First Coastal Bank',
  category: 'mortgage',
  amount_cents: 685000,
  description: 'Mortgage payment',
  day_of_month: 1,
  every_months: 1,
  start_date: '2025-12-01',
  end_date: null,
  generated_through: '2026-07-01',
  next_on: '2026-08-01',
};

describe('RecurringExpenses', () => {
  it('lists what repeats and when it is next due', async () => {
    mockFetch({ 'GET /api/recurring-expenses': { body: [mortgage] } });
    renderWithProviders(<RecurringExpenses />);

    expect(await screen.findByText('Mortgage payment')).toBeInTheDocument();
    expect(screen.getByText('Every month, day 1')).toBeInTheDocument();
    expect(screen.getByText('Aug 1, 2026')).toBeInTheDocument();
    expect(screen.getByText('$6,850.00')).toBeInTheDocument();
  });

  it('adds a quarterly expense', async () => {
    const calls = mockFetch({
      'GET /api/recurring-expenses': { body: [] },
      'GET /api/properties': { body: [{ id: 'p1', name: 'Harbor View Apartments' }] },
      'GET /api/vendors': { body: [] },
      'POST /api/recurring-expenses': { status: 201, body: mortgage },
    });
    renderWithProviders(<RecurringExpenses propertyId="p1" />);

    await userEvent.click(await screen.findByRole('button', { name: 'Set up a recurring expense' }));
    await userEvent.selectOptions(screen.getByLabelText('Category'), 'insurance');
    await userEvent.type(screen.getByLabelText('Amount'), '450');
    await userEvent.selectOptions(screen.getByLabelText('Repeats'), '3');
    const day = screen.getByLabelText('On day');
    await userEvent.clear(day);
    await userEvent.type(day, '15');
    await userEvent.click(screen.getByRole('button', { name: 'Add recurring expense' }));

    const post = calls.find((c) => c.method === 'POST');
    expect(post?.body).toMatchObject({
      property_id: 'p1',
      category: 'insurance',
      amount_cents: 45000,
      every_months: 3,
      day_of_month: 15,
      end_date: null,
    });
  });
});
