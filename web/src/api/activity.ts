import { api } from './client';
import type { Activity } from './types';

export interface ActivityFilter {
  lease_id?: string;
  before?: number;
  limit?: number;
}

export const listActivity = (filter: ActivityFilter = {}) => api<Activity[]>('/activity', { query: { ...filter } });
