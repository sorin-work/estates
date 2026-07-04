import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useState } from 'react';
import {
  createRecurringExpense,
  deleteRecurringExpense,
  listRecurringExpenses,
  updateRecurringExpense,
} from '../api/expenses';
import type { RecurringExpense, RecurringExpenseInput } from '../api/types';
import { formatDate } from '../lib/dates';
import { expenseCategories, recurrences } from '../lib/labels';
import { formatMoney } from '../lib/money';
import RecurringExpenseForm from './forms/RecurringExpenseForm';
import Modal from './Modal';

/** The recurring expenses of one property, or of all of them. */
export default function RecurringExpenses({ propertyId = '' }: { propertyId?: string }) {
  const queryClient = useQueryClient();
  const [form, setForm] = useState<RecurringExpense | 'new' | null>(null);
  const list = useQuery({
    queryKey: ['recurring-expenses', propertyId],
    queryFn: () => listRecurringExpenses(propertyId),
  });

  const refresh = () => {
    for (const key of ['recurring-expenses', 'expenses', 'dashboard']) {
      void queryClient.invalidateQueries({ queryKey: [key] });
    }
  };
  const save = useMutation({
    mutationFn: (input: RecurringExpenseInput) =>
      form && form !== 'new' ? updateRecurringExpense(form.id, input) : createRecurringExpense(input),
    onSuccess: () => {
      refresh();
      setForm(null);
    },
  });
  const remove = useMutation({ mutationFn: deleteRecurringExpense, onSuccess: refresh });

  return (
    <section className="card">
      <div className="card-header">
        <h2>Recurring</h2>
        <button type="button" className="link-button" onClick={() => setForm('new')}>
          Set up a recurring expense
        </button>
      </div>
      {list.data && list.data.length === 0 && (
        <p className="muted">
          Set up mortgage payments, HOA dues or insurance once and they are added as expenses when they fall due.
        </p>
      )}
      {list.data && list.data.length > 0 && (
        <table className="table">
          <thead>
            <tr>
              <th>What</th>
              {!propertyId && <th>Property</th>}
              <th>Repeats</th>
              <th>Next</th>
              <th className="num">Amount</th>
              <th />
            </tr>
          </thead>
          <tbody>
            {list.data.map((r) => (
              <tr key={r.id}>
                <td>
                  {r.description || expenseCategories[r.category]}
                  <div className="muted small">
                    {expenseCategories[r.category]}
                    {r.vendor && ` · ${r.vendor}`}
                  </div>
                </td>
                {!propertyId && <td>{r.property_name}</td>}
                <td>
                  {recurrences[r.every_months]}, day {r.day_of_month}
                </td>
                <td>{r.next_on ? formatDate(r.next_on) : <span className="muted">Ended</span>}</td>
                <td className="num">{formatMoney(r.amount_cents)}</td>
                <td className="row-actions">
                  <button type="button" className="link-button" onClick={() => setForm(r)}>
                    Edit
                  </button>
                  <button
                    type="button"
                    className="link-button danger"
                    onClick={() => {
                      if (confirm('Stop this recurring expense? Expenses already added are kept.')) remove.mutate(r.id);
                    }}
                  >
                    Stop
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
      {form && (
        <Modal
          title={form === 'new' ? 'Add recurring expense' : 'Edit recurring expense'}
          onClose={() => setForm(null)}
        >
          <RecurringExpenseForm
            initial={form === 'new' ? undefined : form}
            propertyId={propertyId || undefined}
            onSubmit={(input) => save.mutateAsync(input)}
            onCancel={() => setForm(null)}
          />
        </Modal>
      )}
    </section>
  );
}
