import { api, download } from './client';
import type { ProfitLoss } from './types';

export interface Period {
  from: string;
  to: string;
}

export const getProfitLoss = (period: Period) => api<ProfitLoss>('/reports/profit-loss', { query: { ...period } });
export const exportProfitLoss = (period: Period) =>
  download('/reports/profit-loss.csv', { ...period }, 'profit-loss.csv');
