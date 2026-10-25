import { useState, type FormEvent } from 'react';
import type { Payment, PaymentInput, PaymentMethod } from '../../api/types';
import { today } from '../../lib/dates';
import { options, paymentMethods } from '../../lib/labels';
import { centsToInput, parseMoney } from '../../lib/money';
import Field from '../Field';
import { FormError } from '../States';
import { useFormSubmit } from '../useFormSubmit';
import MoneyInput from './MoneyInput';

export default function PaymentForm({
  initial,
  suggestedCents,
  onSubmit,
  onCancel,
}: {
  /** A payment being corrected. */
  initial?: Payment;
  /** Pre-filled amount, usually the balance owed. */
  suggestedCents?: number;
  onSubmit: (input: PaymentInput) => Promise<unknown>;
  onCancel: () => void;
}) {
  const [amount, setAmount] = useState(() => {
    if (initial) return centsToInput(initial.amount_cents);
    return suggestedCents && suggestedCents > 0 ? centsToInput(suggestedCents) : '';
  });
  const [paidOn, setPaidOn] = useState(initial?.paid_on ?? today());
  const [method, setMethod] = useState<PaymentMethod>(initial?.method ?? 'bank_transfer');
  const [reference, setReference] = useState(initial?.reference ?? '');
  const [notes, setNotes] = useState(initial?.notes ?? '');
  const { submitting, error, errors, run } = useFormSubmit();

  function submit(e: FormEvent) {
    e.preventDefault();
    const cents = parseMoney(amount);
    const local: Record<string, string> = {};
    if (cents === null || cents === 0) local.amount_cents = 'Enter the amount received';
    void run(() => onSubmit({ amount_cents: cents ?? 0, paid_on: paidOn, method, reference, notes }), local);
  }

  return (
    <form onSubmit={submit} noValidate>
      <FormError error={error} />
      <div className="form-row">
        <Field label="Amount" error={errors.amount_cents}>
          {(id) => <MoneyInput id={id} value={amount} onChange={(e) => setAmount(e.target.value)} />}
        </Field>
        <Field label="Received on" error={errors.paid_on}>
          {(id) => (
            <input id={id} type="date" value={paidOn} max={today()} onChange={(e) => setPaidOn(e.target.value)} />
          )}
        </Field>
      </div>
      <div className="form-row">
        <Field label="Method" error={errors.method}>
          {(id) => (
            <select id={id} value={method} onChange={(e) => setMethod(e.target.value as PaymentMethod)}>
              {options(paymentMethods).map((o) => (
                <option key={o.value} value={o.value}>
                  {o.label}
                </option>
              ))}
            </select>
          )}
        </Field>
        <Field label="Reference" error={errors.reference} hint="Check number, transfer ID…">
          {(id) => <input id={id} value={reference} onChange={(e) => setReference(e.target.value)} />}
        </Field>
      </div>
      <Field label="Notes" error={errors.notes}>
        {(id) => <input id={id} value={notes} onChange={(e) => setNotes(e.target.value)} />}
      </Field>
      <div className="form-actions">
        <button type="button" className="button secondary" onClick={onCancel}>
          Cancel
        </button>
        <button type="submit" className="button" disabled={submitting}>
          {initial ? 'Save payment' : 'Record payment'}
        </button>
      </div>
    </form>
  );
}
