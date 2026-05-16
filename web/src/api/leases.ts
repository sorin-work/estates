import { api } from './client';
import type {
  ChargeInput,
  Deposit,
  DepositInput,
  Lease,
  LeaseInput,
  LeaseStatus,
  Ledger,
  Payment,
  PaymentInput,
} from './types';

export interface LeaseFilter {
  status?: LeaseStatus | '';
  property_id?: string;
  tenant_id?: string;
  unit_id?: string;
}

export const listLeases = (filter: LeaseFilter = {}) => api<Lease[]>('/leases', { query: { ...filter } });
export const getLease = (id: string) => api<Lease>(`/leases/${id}`);
export const getLedger = (id: string) => api<Ledger>(`/leases/${id}/ledger`);
export const createLease = (input: LeaseInput) => api<Lease>('/leases', { method: 'POST', body: input });
export const updateLease = (id: string, input: LeaseInput) =>
  api<Lease>(`/leases/${id}`, { method: 'PUT', body: input });
export const deleteLease = (id: string) => api<void>(`/leases/${id}`, { method: 'DELETE' });
export const terminateLease = (id: string, date: string) =>
  api<Lease>(`/leases/${id}/terminate`, { method: 'POST', body: { date } });

export const addCharge = (leaseId: string, input: ChargeInput) =>
  api<{ id: string }>(`/leases/${leaseId}/charges`, { method: 'POST', body: input });
export const deleteCharge = (id: string) => api<void>(`/charges/${id}`, { method: 'DELETE' });

export const recordPayment = (leaseId: string, input: PaymentInput) =>
  api<Payment>(`/leases/${leaseId}/payments`, { method: 'POST', body: input });

export const getDeposit = (leaseId: string) => api<Deposit>(`/leases/${leaseId}/deposit`);
export const recordDeposit = (leaseId: string, input: DepositInput) =>
  api<Deposit>(`/leases/${leaseId}/deposit`, { method: 'POST', body: input });
export const deleteDepositTransaction = (id: string) => api<void>(`/deposit-transactions/${id}`, { method: 'DELETE' });
