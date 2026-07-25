import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it, vi } from 'vitest';
import type { Lease } from '../../api/types';
import RenewalForm from './RenewalForm';
import RentChangeForm, { percentChange } from './RentChangeForm';

const lease: Lease = {
  id: 'l1',
  unit_id: 'u1',
  unit_label: 'A',
  property_id: 'p1',
  property_name: 'Maple Street Duplex',
  tenant_id: 't1',
  tenant_name: 'Maria Gonzalez',
  start_date: '2026-01-01',
  end_date: '2026-12-31',
  rent_cents: 180000,
  deposit_cents: 180000,
  due_day: 1,
  grace_days: 5,
  late_fee_cents: 7500,
  terminated_on: null,
  notes: '',
  status: 'active',
  current_rent_cents: 180000,
  rent_changes: [],
  renewed_from_id: null,
  renewed_by_id: null,
};

describe('percentChange', () => {
  it('formats the change with a sign', () => {
    expect(percentChange(180000, 185400)).toBe('+3.0%');
    expect(percentChange(100000, 95000)).toBe('-5.0%');
    expect(percentChange(0, 100)).toBe('');
  });
});

describe('RenewalForm', () => {
  it('defaults to another year at the current rent', async () => {
    const onSubmit = vi.fn().mockResolvedValue(undefined);
    render(<RenewalForm lease={lease} onSubmit={onSubmit} onCancel={vi.fn()} />);

    expect(screen.getByText(/starts on Jan 1, 2027/)).toBeInTheDocument();
    expect(screen.getByLabelText('New end date')).toHaveValue('2027-12-31');

    const rent = screen.getByLabelText('Rent');
    await userEvent.clear(rent);
    await userEvent.type(rent, '1854');
    expect(screen.getByText('+3.0%')).toBeInTheDocument();
    await userEvent.click(screen.getByRole('button', { name: 'Renew lease' }));
    expect(onSubmit).toHaveBeenCalledWith({ end_date: '2027-12-31', rent_cents: 185400, notes: '' });
  });
});

describe('RentChangeForm', () => {
  it('submits the new rent from a date', async () => {
    const onSubmit = vi.fn().mockResolvedValue(undefined);
    render(<RentChangeForm lease={lease} onSubmit={onSubmit} onCancel={vi.fn()} />);

    const from = screen.getByLabelText('From');
    await userEvent.clear(from);
    await userEvent.type(from, '2026-09-01');
    const rent = screen.getByLabelText('New rent');
    await userEvent.clear(rent);
    await userEvent.type(rent, '1890');
    await userEvent.click(screen.getByRole('button', { name: 'Save rent change' }));
    expect(onSubmit).toHaveBeenCalledWith({ effective_date: '2026-09-01', rent_cents: 189000, notes: '' });
  });
});
