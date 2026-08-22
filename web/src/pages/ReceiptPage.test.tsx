import { screen } from '@testing-library/react';
import { Route, Routes } from 'react-router';
import { describe, expect, it } from 'vitest';
import { mockFetch, renderWithProviders } from '../test/render';
import ReceiptPage from './ReceiptPage';

const receipt = {
  number: 42,
  issued_on: '2026-09-10',
  payment: {
    id: 'pm1',
    lease_id: 'l1',
    tenant_name: 'Maria Gonzalez',
    property_id: 'p1',
    property_name: 'Maple Street Duplex',
    unit_label: 'A',
    amount_cents: 185400,
    paid_on: '2026-09-01',
    method: 'check',
    reference: '#1042',
    notes: '',
    receipt_number: 42,
  },
  landlord: { name: 'Demo Landlord', email: 'demo@example.com' },
  tenant: { name: 'Maria Gonzalez', email: 'maria@example.com' },
  property: {
    name: 'Maple Street Duplex',
    address_line1: '2218 SE Maple St',
    address_line2: '',
    city: 'Portland',
    region: 'OR',
    postal_code: '97214',
    unit_label: 'A',
  },
  balance_after_cents: 0,
};

describe('ReceiptPage', () => {
  it('prints the payment with who paid and for what', async () => {
    mockFetch({ 'GET /api/payments/pm1/receipt': { body: receipt } });
    renderWithProviders(
      <Routes>
        <Route path="/payments/:id/receipt" element={<ReceiptPage />} />
      </Routes>,
      { route: '/payments/pm1/receipt' },
    );

    expect(await screen.findByRole('heading', { name: 'Rent receipt' })).toBeInTheDocument();
    expect(screen.getByText('No. 00042')).toBeInTheDocument();
    expect(screen.getByText('$1,854.00')).toBeInTheDocument();
    expect(screen.getByText('Check (#1042)')).toBeInTheDocument();
    expect(screen.getByText('2218 SE Maple St, Portland, OR 97214')).toBeInTheDocument();
    expect(screen.getByText('The account was fully paid up after this payment.')).toBeInTheDocument();
  });
});
