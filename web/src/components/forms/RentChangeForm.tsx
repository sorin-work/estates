import { useState, type FormEvent } from 'react';
import type { Lease, RentChangeInput } from '../../api/types';
import { addMonths, today } from '../../lib/dates';
import { centsToInput, formatMoney, parseMoney } from '../../lib/money';
import Field from '../Field';
import { FormError } from '../States';
import { useFormSubmit } from '../useFormSubmit';
import MoneyInput from './MoneyInput';

/** "+4.0%" for the change from one rent to another. */
export function percentChange(from: number, to: number): string {
  if (from <= 0) return '';
  const pct = ((to - from) / from) * 100;
  return `${pct >= 0 ? '+' : ''}${pct.toFixed(1)}%`;
}

export default function RentChangeForm({
  lease,
  onSubmit,
  onCancel,
}: {
  lease: Lease;
  onSubmit: (input: RentChangeInput) => Promise<unknown>;
  onCancel: () => void;
}) {
  // Most places need written notice, so default to the first due date at
  // least two months out.
  const [effective, setEffective] = useState(() => {
    const d = addMonths(today(), 2);
    return `${d.slice(0, 7)}-${String(Math.min(lease.due_day, 28)).padStart(2, '0')}`;
  });
  const [amount, setAmount] = useState(centsToInput(lease.current_rent_cents));
  const [notes, setNotes] = useState('');
  const { submitting, error, errors, run } = useFormSubmit();
  const cents = parseMoney(amount);

  function submit(e: FormEvent) {
    e.preventDefault();
    const local: Record<string, string> = {};
    if (cents === null || cents === 0) local.rent_cents = 'Enter the new rent';
    void run(() => onSubmit({ effective_date: effective, rent_cents: cents ?? 0, notes }), local);
  }

  return (
    <form onSubmit={submit} noValidate>
      <FormError error={error} />
      <p className="muted">
        Rent is {formatMoney(lease.current_rent_cents)} now. Each month is charged at the rent in force on its first
        day, so a change from mid-month applies from the next due date.
      </p>
      <div className="form-row">
        <Field label="From" error={errors.effective_date}>
          {(id) => <input id={id} type="date" value={effective} onChange={(e) => setEffective(e.target.value)} />}
        </Field>
        <Field
          label="New rent"
          error={errors.rent_cents}
          hint={cents ? percentChange(lease.current_rent_cents, cents) : undefined}
        >
          {(id) => <MoneyInput id={id} value={amount} onChange={(e) => setAmount(e.target.value)} />}
        </Field>
      </div>
      <Field label="Notes" error={errors.notes}>
        {(id) => (
          <input id={id} value={notes} placeholder="e.g. notice sent on…" onChange={(e) => setNotes(e.target.value)} />
        )}
      </Field>
      <div className="form-actions">
        <button type="button" className="button secondary" onClick={onCancel}>
          Cancel
        </button>
        <button type="submit" className="button" disabled={submitting}>
          Save rent change
        </button>
      </div>
    </form>
  );
}
