import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useState } from 'react';
import { Link, useSearchParams } from 'react-router';
import { createExpense, deleteExpense, listExpenses, updateExpense } from '../api/expenses';
import { listProperties } from '../api/properties';
import type { Expense, ExpenseCategory, ExpenseInput } from '../api/types';
import ExpenseForm from '../components/forms/ExpenseForm';
import Modal from '../components/Modal';
import PageHeader from '../components/PageHeader';
import { EmptyState, ErrorState, Loading } from '../components/States';
import { formatDate, today } from '../lib/dates';
import { expenseCategories, options } from '../lib/labels';
import { formatMoney } from '../lib/money';

export default function ExpensesPage() {
  const queryClient = useQueryClient();
  const [params, setParams] = useSearchParams();
  const year = today().slice(0, 4);
  const propertyId = params.get('property_id') ?? '';
  const category = (params.get('category') ?? '') as ExpenseCategory | '';
  const from = params.get('from') ?? `${year}-01-01`;
  const to = params.get('to') ?? `${year}-12-31`;
  const [form, setForm] = useState<Expense | 'new' | null>(null);

  const filter = { property_id: propertyId, category, from, to };
  const expenses = useQuery({ queryKey: ['expenses', filter], queryFn: () => listExpenses(filter) });
  const properties = useQuery({ queryKey: ['properties'], queryFn: listProperties });

  const refresh = () => {
    for (const key of ['expenses', 'maintenance', 'dashboard']) {
      void queryClient.invalidateQueries({ queryKey: [key] });
    }
  };
  const save = useMutation({
    mutationFn: (input: ExpenseInput) =>
      form && form !== 'new' ? updateExpense(form.id, input) : createExpense(input),
    onSuccess: () => {
      refresh();
      setForm(null);
    },
  });
  const remove = useMutation({ mutationFn: deleteExpense, onSuccess: refresh });

  function setFilter(key: string, value: string) {
    const next = new URLSearchParams(params);
    if (value) next.set(key, value);
    else next.delete(key);
    setParams(next, { replace: true });
  }

  const data = expenses.data;
  const top = data?.by_category[0]?.amount_cents ?? 0;

  return (
    <>
      <PageHeader
        title="Expenses"
        actions={
          <button type="button" className="button" onClick={() => setForm('new')}>
            Add expense
          </button>
        }
      />
      <div className="toolbar">
        <select aria-label="Property" value={propertyId} onChange={(e) => setFilter('property_id', e.target.value)}>
          <option value="">All properties</option>
          {properties.data?.map((p) => (
            <option key={p.id} value={p.id}>
              {p.name}
            </option>
          ))}
        </select>
        <select aria-label="Category" value={category} onChange={(e) => setFilter('category', e.target.value)}>
          <option value="">All categories</option>
          {options(expenseCategories).map((o) => (
            <option key={o.value} value={o.value}>
              {o.label}
            </option>
          ))}
        </select>
        <label className="inline">
          From <input type="date" value={from} onChange={(e) => setFilter('from', e.target.value)} />
        </label>
        <label className="inline">
          To <input type="date" value={to} onChange={(e) => setFilter('to', e.target.value)} />
        </label>
      </div>
      {expenses.isPending && <Loading />}
      {expenses.error && <ErrorState error={expenses.error} />}
      {data && data.expenses.length === 0 && (
        <EmptyState title="No expenses in this period">
          Record repairs, insurance, taxes and other costs to see what each property really earns.
        </EmptyState>
      )}
      {data && data.expenses.length > 0 && (
        <div className="grid-sidebar">
          <div className="card flush">
            <table className="table">
              <thead>
                <tr>
                  <th>Date</th>
                  <th>Property</th>
                  <th>Category</th>
                  <th>Details</th>
                  <th className="num">Amount</th>
                  <th />
                </tr>
              </thead>
              <tbody>
                {data.expenses.map((e) => (
                  <tr key={e.id}>
                    <td>{formatDate(e.spent_on)}</td>
                    <td>{e.property_name}</td>
                    <td>{expenseCategories[e.category]}</td>
                    <td>
                      {e.vendor &&
                        (e.vendor_id ? (
                          <Link to={`/vendors/${e.vendor_id}`}>
                            <strong>{e.vendor}</strong>{' '}
                          </Link>
                        ) : (
                          <strong>{e.vendor} </strong>
                        ))}
                      <span className="muted">{e.description}</span>
                      {e.maintenance_title && <div className="muted small">For: {e.maintenance_title}</div>}
                    </td>
                    <td className="num">{formatMoney(e.amount_cents)}</td>
                    <td className="row-actions">
                      <button type="button" className="link-button" onClick={() => setForm(e)}>
                        Edit
                      </button>
                      <button
                        type="button"
                        className="link-button danger"
                        onClick={() => {
                          if (confirm('Delete this expense?')) remove.mutate(e.id);
                        }}
                      >
                        Delete
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          <aside className="card">
            <div className="stat-label">Total</div>
            <div className="stat-value">{formatMoney(data.total_cents)}</div>
            <ul className="bars">
              {data.by_category.map((c) => (
                <li key={c.category}>
                  <div className="bar-label">
                    <span>{expenseCategories[c.category]}</span>
                    <span>{formatMoney(c.amount_cents)}</span>
                  </div>
                  <div className="bar-track">
                    <div className="bar-fill" style={{ width: `${(c.amount_cents / top) * 100}%` }} />
                  </div>
                </li>
              ))}
            </ul>
          </aside>
        </div>
      )}
      {form && (
        <Modal title={form === 'new' ? 'Add expense' : 'Edit expense'} onClose={() => setForm(null)}>
          <ExpenseForm
            initial={form === 'new' ? undefined : form}
            propertyId={propertyId || undefined}
            onSubmit={(input) => save.mutateAsync(input)}
            onCancel={() => setForm(null)}
          />
        </Modal>
      )}
    </>
  );
}
