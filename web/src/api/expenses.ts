import { api } from './client';
import type { ExpenseCategory, Expense, ExpenseInput, ExpenseList } from './types';

export interface ExpenseFilter {
  property_id?: string;
  category?: ExpenseCategory | '';
  from?: string;
  to?: string;
}

export const listExpenses = (filter: ExpenseFilter = {}) => api<ExpenseList>('/expenses', { query: { ...filter } });
export const createExpense = (input: ExpenseInput) => api<Expense>('/expenses', { method: 'POST', body: input });
export const updateExpense = (id: string, input: ExpenseInput) =>
  api<Expense>(`/expenses/${id}`, { method: 'PUT', body: input });
export const deleteExpense = (id: string) => api<void>(`/expenses/${id}`, { method: 'DELETE' });
