import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it, vi } from 'vitest';
import { ApiError } from '../../api/client';
import { today } from '../../lib/dates';
import PaymentForm from './PaymentForm';

describe('PaymentForm', () => {
  it('pre-fills the balance and submits cents', async () => {
    const onSubmit = vi.fn().mockResolvedValue(undefined);
    render(<PaymentForm suggestedCents={182500} onSubmit={onSubmit} onCancel={() => {}} />);

    expect(screen.getByLabelText('Amount')).toHaveValue('1825.00');
    await userEvent.selectOptions(screen.getByLabelText('Method'), 'check');
    await userEvent.type(screen.getByLabelText('Reference'), '#1042');
    await userEvent.click(screen.getByRole('button', { name: 'Record payment' }));

    expect(onSubmit).toHaveBeenCalledWith({
      amount_cents: 182500,
      paid_on: today(),
      method: 'check',
      reference: '#1042',
      notes: '',
    });
  });

  it('does not submit an amount it cannot read', async () => {
    const onSubmit = vi.fn();
    render(<PaymentForm onSubmit={onSubmit} onCancel={() => {}} />);

    await userEvent.type(screen.getByLabelText('Amount'), '12.345');
    await userEvent.click(screen.getByRole('button', { name: 'Record payment' }));

    expect(onSubmit).not.toHaveBeenCalled();
    expect(screen.getByText('Enter the amount received')).toBeInTheDocument();
  });

  it('shows field errors from the API', async () => {
    const onSubmit = vi
      .fn()
      .mockRejectedValue(
        new ApiError(422, 'validation_failed', 'some fields are invalid', { paid_on: 'cannot be in the future' }),
      );
    render(<PaymentForm suggestedCents={1000} onSubmit={onSubmit} onCancel={() => {}} />);

    await userEvent.click(screen.getByRole('button', { name: 'Record payment' }));

    expect(await screen.findByText('cannot be in the future')).toBeInTheDocument();
    expect(screen.getByRole('alert')).toHaveTextContent('Please fix the highlighted fields.');
  });
});
