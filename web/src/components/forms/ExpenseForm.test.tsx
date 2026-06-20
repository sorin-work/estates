import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it, vi } from 'vitest';
import { mockFetch, renderWithProviders } from '../../test/render';
import ExpenseForm from './ExpenseForm';

describe('ExpenseForm', () => {
  it('links a saved vendor, or keeps a typed name', async () => {
    mockFetch({
      'GET /api/properties': { body: [{ id: 'p1', name: 'Maple Street Duplex' }] },
      'GET /api/vendors': { body: [{ id: 'v1', name: 'Rapid Rooter Plumbing' }] },
      'GET /api/maintenance': { body: [] },
    });
    const onSubmit = vi.fn().mockResolvedValue(undefined);
    renderWithProviders(<ExpenseForm propertyId="p1" onSubmit={onSubmit} onCancel={vi.fn()} />);

    await userEvent.type(screen.getByLabelText('Amount'), '185');
    await userEvent.type(await screen.findByLabelText('Vendor name'), 'Hardware Depot');
    await userEvent.click(screen.getByRole('button', { name: 'Add expense' }));
    expect(onSubmit).toHaveBeenLastCalledWith(
      expect.objectContaining({ vendor: 'Hardware Depot', vendor_id: null, amount_cents: 18500 }),
    );

    await screen.findByRole('option', { name: 'Rapid Rooter Plumbing' });
    await userEvent.selectOptions(screen.getByLabelText('Vendor'), 'v1');
    expect(screen.queryByLabelText('Vendor name')).not.toBeInTheDocument();
    await userEvent.click(screen.getByRole('button', { name: 'Add expense' }));
    expect(onSubmit).toHaveBeenLastCalledWith(
      expect.objectContaining({ vendor: 'Rapid Rooter Plumbing', vendor_id: 'v1' }),
    );
  });
});
