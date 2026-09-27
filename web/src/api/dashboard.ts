import { api } from './client';
import type { Dashboard, Reminder } from './types';

export const getDashboard = () => api<Dashboard>('/dashboard');
export const getReminders = () => api<Reminder[]>('/reminders');
