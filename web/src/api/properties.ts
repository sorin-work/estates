import { api } from './client';
import type { Property, PropertyInput, Unit, UnitInput } from './types';

export const listProperties = () => api<Property[]>('/properties');
export const getProperty = (id: string) => api<Property>(`/properties/${id}`);
export const createProperty = (input: PropertyInput) => api<Property>('/properties', { method: 'POST', body: input });
export const updateProperty = (id: string, input: PropertyInput) =>
  api<Property>(`/properties/${id}`, { method: 'PUT', body: input });
export const deleteProperty = (id: string) => api<void>(`/properties/${id}`, { method: 'DELETE' });

export const listUnits = (propertyId?: string) => api<Unit[]>('/units', { query: { property_id: propertyId } });
export const createUnit = (propertyId: string, input: UnitInput) =>
  api<Unit>(`/properties/${propertyId}/units`, { method: 'POST', body: input });
export const updateUnit = (id: string, input: UnitInput) => api<Unit>(`/units/${id}`, { method: 'PUT', body: input });
export const deleteUnit = (id: string) => api<void>(`/units/${id}`, { method: 'DELETE' });
