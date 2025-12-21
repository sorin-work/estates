import { api } from './client';
import type { Tenant, TenantInput } from './types';

export const listTenants = (q?: string) => api<Tenant[]>('/tenants', { query: { q } });
export const getTenant = (id: string) => api<Tenant>(`/tenants/${id}`);
export const createTenant = (input: TenantInput) => api<Tenant>('/tenants', { method: 'POST', body: input });
export const updateTenant = (id: string, input: TenantInput) =>
  api<Tenant>(`/tenants/${id}`, { method: 'PUT', body: input });
export const deleteTenant = (id: string) => api<void>(`/tenants/${id}`, { method: 'DELETE' });
