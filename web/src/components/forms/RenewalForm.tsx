import { useState, type FormEvent } from 'react';
import type { Lease, RenewalInput } from '../../api/types';
import { addDays, addMonths, formatDate } from '../../lib/dates';
import { centsToInput, parseMoney } from '../../lib/money';
import Field from '../Field';
import { FormError } from '../States';
import { useFormSubmit } from '../useFormSubmit';
import { percentChange } from './RentChangeForm';
import MoneyInput from './MoneyInput';

export default function RenewalForm({
  lease,
  onSubmit,
  onCancel,
}: {
  lease: Lease;
  onSubmit: (input: RenewalInput) => Promise<unknown>;
  onCancel: () => void;
}) {
  const start = addDays(lease.end_date, 1);
  const [end, setEnd] = useState(addDays(addMonths(start, 12), -1));
  const [amount, setAmount] = useState(centsToInput(lease.current_rent_cents));
  const [notes, setNotes] = useState('');
  const { submitting, error, errors, run } = useFormSubmit();
  const cents = parseMoney(amount);

  function submit(e: FormEvent) {
    e.preventDefault();
    const local: Record<string, string> = {};
    if (cents === null || cents === 0) local.rent_cents = 'Enter the rent';
    void run(() => onSubmit({ end_date: end, rent_cents: cents ?? 0, notes }), local);
  }

  return (
    <form onSubmit={submit} noValidate>
      <FormError error={error} />
      <p className="muted">
        The renewal starts on {formatDate(start)} with the same due day, grace period and late fee. The deposit moves to
        the new lease.
      </p>
      <div className="form-row">
        <Field label="New end date" error={errors.end_date}>
          {(id) => <input id={id} type="date" value={end} min={start} onChange={(e) => setEnd(e.target.value)} />}
        </Field>
        <Field
          label="Rent"
          error={errors.rent_cents}
          hint={cents ? percentChange(lease.current_rent_cents, cents) : undefined}
        >
          {(id) => <MoneyInput id={id} value={amount} onChange={(e) => setAmount(e.target.value)} />}
        </Field>
      </div>
      <Field label="Notes" error={errors.notes}>
        {(id) => <textarea id={id} rows={2} value={notes} onChange={(e) => setNotes(e.target.value)} />}
      </Field>
      <div className="form-actions">
        <button type="button" className="button secondary" onClick={onCancel}>
          Cancel
        </button>
        <button type="submit" className="button" disabled={submitting}>
          Renew lease
        </button>
      </div>
    </form>
  );
}
