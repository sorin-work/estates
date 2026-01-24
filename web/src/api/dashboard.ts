import { api } from './client';
import type { Dashboard } from './types';

export const getDashboard = () => api<Dashboard>('/dashboard');
