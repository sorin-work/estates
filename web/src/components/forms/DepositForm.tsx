import { useState, type FormEvent } from 'react';
import type { Deposit, DepositInput, DepositKind } from '../../api/types';
import { today } from '../../lib/dates';
import { depositKinds, options } from '../../lib/labels';
import { centsToInput, formatMoney, parseMoney } from '../../lib/money';
import Field from '../Field';
import { FormError } from '../States';
import { useFormSubmit } from '../useFormSubmit';
import MoneyInput from './MoneyInput';

/** What is left to collect when receiving, or what is held when paying out. */
function suggested(deposit: Deposit, kind: DepositKind): number {
  if (kind === 'received') return Math.max(deposit.agreed_cents - deposit.received_cents, 0);
  return deposit.held_cents;
}

export default function DepositForm({
  deposit,
  onSubmit,
  onCancel,
}: {
  deposit: Deposit;
  onSubmit: (input: DepositInput) => Promise<unknown>;
  onCancel: () => void;
}) {
  const [kind, setKind] = useState<DepositKind>(deposit.held_cents > 0 ? 'returned' : 'received');
  const [amount, setAmount] = useState(centsToInput(suggested(deposit, kind)));
  const [occurredOn, setOccurredOn] = useState(today());
  const [notes, setNotes] = useState('');
  const { submitting, error, errors, run } = useFormSubmit();

  function changeKind(next: DepositKind) {
    setKind(next);
    setAmount(centsToInput(suggested(deposit, next)));
  }

  function submit(e: FormEvent) {
    e.preventDefault();
    const cents = parseMoney(amount);
    const local: Record<string, string> = {};
    if (cents === null || cents === 0) local.amount_cents = 'Enter an amount';
    else if (kind !== 'received' && cents > deposit.held_cents)
      local.amount_cents = `Only ${formatMoney(deposit.held_cents)} is held`;
    void run(() => onSubmit({ kind, amount_cents: cents ?? 0, occurred_on: occurredOn, notes }), local);
  }

  return (
    <form onSubmit={submit} noValidate>
      <FormError error={error} />
      <p className="muted">
        {formatMoney(deposit.held_cents)} held of {formatMoney(deposit.agreed_cents)} agreed in the lease.
      </p>
      <div className="form-row">
        <Field label="Type" error={errors.kind}>
          {(id) => (
            <select id={id} value={kind} onChange={(e) => changeKind(e.target.value as DepositKind)}>
              {options(depositKinds).map((o) => (
                <option key={o.value} value={o.value}>
                  {o.label}
                </option>
              ))}
            </select>
          )}
        </Field>
        <Field label="Date" error={errors.occurred_on}>
          {(id) => (
            <input
              id={id}
              type="date"
              value={occurredOn}
              max={today()}
              onChange={(e) => setOccurredOn(e.target.value)}
            />
          )}
        </Field>
      </div>
      <Field label="Amount" error={errors.amount_cents}>
        {(id) => <MoneyInput id={id} value={amount} onChange={(e) => setAmount(e.target.value)} />}
      </Field>
      <Field
        label="Notes"
        error={errors.notes}
        hint={kind === 'withheld' ? 'What it covers, e.g. cleaning or unpaid rent.' : undefined}
      >
        {(id) => <input id={id} value={notes} onChange={(e) => setNotes(e.target.value)} />}
      </Field>
      <div className="form-actions">
        <button type="button" className="button secondary" onClick={onCancel}>
          Cancel
        </button>
        <button type="submit" className="button" disabled={submitting}>
          Save
        </button>
      </div>
    </form>
  );
}
