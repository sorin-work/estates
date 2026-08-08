import { api, download } from './client';
import type { Balance, PaymentList } from './types';

export interface PaymentFilter {
  lease_id?: string;
  property_id?: string;
  from?: string;
  to?: string;
  limit?: number;
  offset?: number;
}

export const listPayments = (filter: PaymentFilter = {}) => api<PaymentList>('/payments', { query: { ...filter } });
export const exportPayments = (filter: Omit<PaymentFilter, 'limit' | 'offset'>) =>
  download('/payments/export.csv', { ...filter }, 'payments.csv');
export const deletePayment = (id: string) => api<void>(`/payments/${id}`, { method: 'DELETE' });

export const listBalances = (overdueOnly = false) =>
  api<Balance[]>('/rent/balances', { query: { overdue: overdueOnly || undefined } });
export const runBilling = () =>
  api<{ leases: number; rent_charges: number; late_fees: number }>('/rent/run', { method: 'POST' });
