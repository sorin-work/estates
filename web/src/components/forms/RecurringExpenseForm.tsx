import { useQuery } from '@tanstack/react-query';
import { useState, type FormEvent } from 'react';
import { listProperties } from '../../api/properties';
import type { ExpenseCategory, RecurrenceMonths, RecurringExpense, RecurringExpenseInput } from '../../api/types';
import { today } from '../../lib/dates';
import { expenseCategories, options, recurrences } from '../../lib/labels';
import { centsToInput, parseMoney } from '../../lib/money';
import Field from '../Field';
import { FormError } from '../States';
import { useFormSubmit } from '../useFormSubmit';
import MoneyInput from './MoneyInput';
import VendorPicker, { type VendorChoice } from './VendorPicker';

const intervals: RecurrenceMonths[] = [1, 3, 6, 12];

export default function RecurringExpenseForm({
  initial,
  propertyId,
  onSubmit,
  onCancel,
}: {
  initial?: RecurringExpense;
  propertyId?: string;
  onSubmit: (input: RecurringExpenseInput) => Promise<unknown>;
  onCancel: () => void;
}) {
  const [property, setProperty] = useState(initial?.property_id ?? propertyId ?? '');
  const [category, setCategory] = useState<ExpenseCategory>(initial?.category ?? 'mortgage');
  const [amount, setAmount] = useState(centsToInput(initial?.amount_cents ?? 0));
  const [description, setDescription] = useState(initial?.description ?? '');
  const [vendor, setVendor] = useState<VendorChoice>({
    vendorId: initial?.vendor_id ?? '',
    name: initial?.vendor ?? '',
  });
  const [every, setEvery] = useState<RecurrenceMonths>(initial?.every_months ?? 1);
  const [day, setDay] = useState(String(initial?.day_of_month ?? 1));
  const [start, setStart] = useState(initial?.start_date ?? today());
  const [end, setEnd] = useState(initial?.end_date ?? '');
  const properties = useQuery({ queryKey: ['properties'], queryFn: listProperties });
  const { submitting, error, errors, run } = useFormSubmit();

  function submit(e: FormEvent) {
    e.preventDefault();
    const cents = parseMoney(amount);
    const local: Record<string, string> = {};
    if (cents === null || cents === 0) local.amount_cents = 'Enter the amount';
    void run(
      () =>
        onSubmit({
          property_id: property,
          category,
          amount_cents: cents ?? 0,
          description,
          vendor: vendor.name,
          vendor_id: vendor.vendorId || null,
          day_of_month: Number(day),
          every_months: every,
          start_date: start,
          end_date: end || null,
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
            <select id={id} value={property} onChange={(e) => setProperty(e.target.value)}>
              <option value="">Choose…</option>
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
        <Field label="Description" error={errors.description}>
          {(id) => <input id={id} value={description} onChange={(e) => setDescription(e.target.value)} />}
        </Field>
      </div>
      <div className="form-row">
        <VendorPicker value={vendor} onChange={setVendor} error={errors.vendor_id ?? errors.vendor} />
      </div>
      <div className="form-row">
        <Field label="Repeats" error={errors.every_months}>
          {(id) => (
            <select id={id} value={every} onChange={(e) => setEvery(Number(e.target.value) as RecurrenceMonths)}>
              {intervals.map((n) => (
                <option key={n} value={n}>
                  {recurrences[n]}
                </option>
              ))}
            </select>
          )}
        </Field>
        <Field label="On day" error={errors.day_of_month} hint="31 means the last day of the month">
          {(id) => (
            <input id={id} type="number" min={1} max={31} value={day} onChange={(e) => setDay(e.target.value)} />
          )}
        </Field>
      </div>
      <div className="form-row">
        <Field label="Starts" error={errors.start_date}>
          {(id) => <input id={id} type="date" value={start} onChange={(e) => setStart(e.target.value)} />}
        </Field>
        <Field label="Ends" error={errors.end_date} hint="Leave empty if it keeps going">
          {(id) => <input id={id} type="date" value={end} onChange={(e) => setEnd(e.target.value)} />}
        </Field>
      </div>
      <div className="form-actions">
        <button type="button" className="button secondary" onClick={onCancel}>
          Cancel
        </button>
        <button type="submit" className="button" disabled={submitting}>
          {initial ? 'Save' : 'Add recurring expense'}
        </button>
      </div>
    </form>
  );
}
