import { useQuery } from '@tanstack/react-query';
import { useState, type FormEvent } from 'react';
import { listMaintenance } from '../../api/maintenance';
import { listProperties } from '../../api/properties';
import type { Expense, ExpenseCategory, ExpenseInput } from '../../api/types';
import { today } from '../../lib/dates';
import { expenseCategories, options } from '../../lib/labels';
import { centsToInput, parseMoney } from '../../lib/money';
import Field from '../Field';
import { FormError } from '../States';
import { useFormSubmit } from '../useFormSubmit';
import MoneyInput from './MoneyInput';

export default function ExpenseForm({
  initial,
  propertyId,
  onSubmit,
  onCancel,
}: {
  initial?: Expense;
  propertyId?: string;
  onSubmit: (input: ExpenseInput) => Promise<unknown>;
  onCancel: () => void;
}) {
  const [property, setProperty] = useState(initial?.property_id ?? propertyId ?? '');
  const [category, setCategory] = useState<ExpenseCategory>(initial?.category ?? 'repairs');
  const [amount, setAmount] = useState(centsToInput(initial?.amount_cents ?? 0));
  const [spentOn, setSpentOn] = useState(initial?.spent_on ?? today());
  const [vendor, setVendor] = useState(initial?.vendor ?? '');
  const [description, setDescription] = useState(initial?.description ?? '');
  const [requestId, setRequestId] = useState(initial?.maintenance_request_id ?? '');
  const properties = useQuery({ queryKey: ['properties'], queryFn: listProperties });
  const requests = useQuery({
    queryKey: ['maintenance', { property_id: property }],
    queryFn: () => listMaintenance({ property_id: property }),
    enabled: property !== '',
  });
  const { submitting, error, errors, run } = useFormSubmit();

  function submit(e: FormEvent) {
    e.preventDefault();
    const cents = parseMoney(amount);
    const local: Record<string, string> = {};
    if (cents === null || cents === 0) local.amount_cents = 'Enter the amount spent';
    void run(
      () =>
        onSubmit({
          property_id: property,
          maintenance_request_id: requestId || null,
          category,
          amount_cents: cents ?? 0,
          spent_on: spentOn,
          vendor,
          description,
        }),
      local,
    );
  }

  return (
    <form onSubmit={submit} noValidate>
      <FormError error={error} />
      <div className="form-row">
        <Field label="Property" error={errors.property_id}>
          {(id) => (
            <select
              id={id}
              value={property}
              onChange={(e) => {
                setProperty(e.target.value);
                setRequestId('');
              }}
            >
              <option value="">Choose a property…</option>
              {properties.data?.map((p) => (
                <option key={p.id} value={p.id}>
                  {p.name}
                </option>
              ))}
            </select>
          )}
        </Field>
        <Field label="Category" error={errors.category}>
          {(id) => (
            <select id={id} value={category} onChange={(e) => setCategory(e.target.value as ExpenseCategory)}>
              {options(expenseCategories).map((o) => (
                <option key={o.value} value={o.value}>
                  {o.label}
                </option>
              ))}
            </select>
          )}
        </Field>
      </div>
      <div className="form-row">
        <Field label="Amount" error={errors.amount_cents}>
          {(id) => <MoneyInput id={id} value={amount} onChange={(e) => setAmount(e.target.value)} />}
        </Field>
        <Field label="Date" error={errors.spent_on}>
          {(id) => <input id={id} type="date" value={spentOn} onChange={(e) => setSpentOn(e.target.value)} />}
        </Field>
      </div>
      <div className="form-row">
        <Field label="Vendor" error={errors.vendor}>
          {(id) => <input id={id} value={vendor} onChange={(e) => setVendor(e.target.value)} />}
        </Field>
        <Field label="Maintenance request" error={errors.maintenance_request_id}>
          {(id) => (
            <select id={id} value={requestId} disabled={!property} onChange={(e) => setRequestId(e.target.value)}>
              <option value="">None</option>
              {requests.data?.map((r) => (
                <option key={r.id} value={r.id}>
                  {r.title}
                </option>
              ))}
            </select>
          )}
        </Field>
      </div>
      <Field label="Description" error={errors.description}>
        {(id) => <input id={id} value={description} onChange={(e) => setDescription(e.target.value)} />}
      </Field>
      <div className="form-actions">
        <button type="button" className="button secondary" onClick={onCancel}>
          Cancel
        </button>
        <button type="submit" className="button" disabled={submitting}>
          {initial ? 'Save expense' : 'Add expense'}
        </button>
      </div>
    </form>
  );
}
