import { api } from './client';
import type { Vendor, VendorInput, VendorTrade } from './types';

export const listVendors = (trade: VendorTrade | '' = '') => api<Vendor[]>('/vendors', { query: { trade } });
export const getVendor = (id: string) => api<Vendor>(`/vendors/${id}`);
export const createVendor = (input: VendorInput) => api<Vendor>('/vendors', { method: 'POST', body: input });
export const updateVendor = (id: string, input: VendorInput) =>
  api<Vendor>(`/vendors/${id}`, { method: 'PUT', body: input });
export const deleteVendor = (id: string) => api<void>(`/vendors/${id}`, { method: 'DELETE' });
