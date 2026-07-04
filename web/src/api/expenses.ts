import { api } from './client';
import type {
  Expense,
  ExpenseCategory,
  ExpenseInput,
  ExpenseList,
  RecurringExpense,
  RecurringExpenseInput,
} from './types';

export interface ExpenseFilter {
  property_id?: string;
  category?: ExpenseCategory | '';
  vendor_id?: string;
  from?: string;
  to?: string;
}

export const listExpenses = (filter: ExpenseFilter = {}) => api<ExpenseList>('/expenses', { query: { ...filter } });
export const createExpense = (input: ExpenseInput) => api<Expense>('/expenses', { method: 'POST', body: input });
export const updateExpense = (id: string, input: ExpenseInput) =>
  api<Expense>(`/expenses/${id}`, { method: 'PUT', body: input });
export const deleteExpense = (id: string) => api<void>(`/expenses/${id}`, { method: 'DELETE' });

export const listRecurringExpenses = (propertyId = '') =>
  api<RecurringExpense[]>('/recurring-expenses', { query: { property_id: propertyId } });
export const createRecurringExpense = (input: RecurringExpenseInput) =>
  api<RecurringExpense>('/recurring-expenses', { method: 'POST', body: input });
export const updateRecurringExpense = (id: string, input: RecurringExpenseInput) =>
  api<RecurringExpense>(`/recurring-expenses/${id}`, { method: 'PUT', body: input });
export const deleteRecurringExpense = (id: string) => api<void>(`/recurring-expenses/${id}`, { method: 'DELETE' });
