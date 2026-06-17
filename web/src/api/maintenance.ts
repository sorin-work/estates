import { api } from './client';
import type { MaintenanceInput, MaintenancePriority, MaintenanceRequest, MaintenanceStatus } from './types';

export interface MaintenanceFilter {
  property_id?: string;
  status?: MaintenanceStatus | 'active' | '';
  priority?: MaintenancePriority | '';
  vendor_id?: string;
}

export const listMaintenance = (filter: MaintenanceFilter = {}) =>
  api<MaintenanceRequest[]>('/maintenance', { query: { ...filter } });
export const createMaintenance = (input: MaintenanceInput) =>
  api<MaintenanceRequest>('/maintenance', { method: 'POST', body: input });
export const updateMaintenance = (id: string, input: MaintenanceInput) =>
  api<MaintenanceRequest>(`/maintenance/${id}`, { method: 'PUT', body: input });
export const setMaintenanceStatus = (id: string, status: MaintenanceStatus) =>
  api<MaintenanceRequest>(`/maintenance/${id}/status`, { method: 'POST', body: { status } });
export const deleteMaintenance = (id: string) => api<void>(`/maintenance/${id}`, { method: 'DELETE' });
