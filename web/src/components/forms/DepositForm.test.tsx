import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it, vi } from 'vitest';
import type { Deposit } from '../../api/types';
import DepositForm from './DepositForm';

const base: Deposit = {
  lease_id: 'l1',
  agreed_cents: 150000,
  received_cents: 0,
  returned_cents: 0,
  withheld_cents: 0,
  held_cents: 0,
  transactions: [],
};

describe('DepositForm', () => {
  it('suggests the agreed deposit when nothing is held', () => {
    render(<DepositForm deposit={base} onSubmit={vi.fn()} onCancel={vi.fn()} />);
    expect(screen.getByLabelText('Type')).toHaveValue('received');
    expect(screen.getByLabelText('Amount')).toHaveValue('1500.00');
  });

  it('returns what is held and refuses more', async () => {
    const onSubmit = vi.fn().mockResolvedValue(undefined);
    const held = { ...base, received_cents: 150000, held_cents: 150000 };
    render(<DepositForm deposit={held} onSubmit={onSubmit} onCancel={vi.fn()} />);
    expect(screen.getByLabelText('Type')).toHaveValue('returned');

    const amount = screen.getByLabelText('Amount');
    await userEvent.clear(amount);
    await userEvent.type(amount, '1600');
    await userEvent.click(screen.getByRole('button', { name: 'Save' }));
    expect(screen.getByText('Only $1,500.00 is held')).toBeInTheDocument();
    expect(onSubmit).not.toHaveBeenCalled();

    await userEvent.selectOptions(screen.getByLabelText('Type'), 'withheld');
    await userEvent.clear(amount);
    await userEvent.type(amount, '325');
    await userEvent.type(screen.getByLabelText('Notes'), 'Cleaning');
    await userEvent.click(screen.getByRole('button', { name: 'Save' }));
    expect(onSubmit).toHaveBeenCalledWith(
      expect.objectContaining({ kind: 'withheld', amount_cents: 32500, notes: 'Cleaning' }),
    );
  });
});
