import { useState, type FormEvent } from 'react';
import type { ChargeInput } from '../../api/types';
import { today } from '../../lib/dates';
import { chargeKinds } from '../../lib/labels';
import { parseMoney } from '../../lib/money';
import Field from '../Field';
import { FormError } from '../States';
import { useFormSubmit } from '../useFormSubmit';
import MoneyInput from './MoneyInput';

const kinds: ChargeInput['kind'][] = ['utility', 'late_fee', 'other'];

export default function ChargeForm({
  onSubmit,
  onCancel,
}: {
  onSubmit: (input: ChargeInput) => Promise<unknown>;
  onCancel: () => void;
}) {
  const [kind, setKind] = useState<ChargeInput['kind']>('utility');
  const [description, setDescription] = useState('');
  const [amount, setAmount] = useState('');
  const [dueDate, setDueDate] = useState(today());
  const { submitting, error, errors, run } = useFormSubmit();

  function submit(e: FormEvent) {
    e.preventDefault();
    const cents = parseMoney(amount);
    const local: Record<string, string> = {};
    if (cents === null || cents === 0) local.amount_cents = 'Enter an amount';
    void run(() => onSubmit({ kind, description, amount_cents: cents ?? 0, due_date: dueDate }), local);
  }

  return (
    <form onSubmit={submit} noValidate>
      <FormError error={error} />
      <div className="form-row">
        <Field label="Type" error={errors.kind}>
          {(id) => (
            <select id={id} value={kind} onChange={(e) => setKind(e.target.value as ChargeInput['kind'])}>
              {kinds.map((k) => (
                <option key={k} value={k}>
                  {chargeKinds[k]}
                </option>
              ))}
            </select>
          )}
        </Field>
        <Field label="Due date" error={errors.due_date}>
          {(id) => <input id={id} type="date" value={dueDate} onChange={(e) => setDueDate(e.target.value)} />}
        </Field>
      </div>
      <Field label="Amount" error={errors.amount_cents}>
        {(id) => <MoneyInput id={id} value={amount} onChange={(e) => setAmount(e.target.value)} />}
      </Field>
      <Field label="Description" error={errors.description}>
        {(id) => (
          <input
            id={id}
            value={description}
            placeholder="e.g. Water bill, March"
            onChange={(e) => setDescription(e.target.value)}
          />
        )}
      </Field>
      <div className="form-actions">
        <button type="button" className="button secondary" onClick={onCancel}>
          Cancel
        </button>
        <button type="submit" className="button" disabled={submitting}>
          Add charge
        </button>
      </div>
    </form>
  );
}
