import { useState, type FormEvent } from 'react';
import type { Unit, UnitInput } from '../../api/types';
import { centsToInput, parseMoney } from '../../lib/money';
import Field from '../Field';
import { FormError } from '../States';
import { useFormSubmit } from '../useFormSubmit';
import MoneyInput from './MoneyInput';

export default function UnitForm({
  initial,
  onSubmit,
  onCancel,
}: {
  initial?: Unit;
  onSubmit: (input: UnitInput) => Promise<unknown>;
  onCancel: () => void;
}) {
  const [label, setLabel] = useState(initial?.label ?? '');
  const [bedrooms, setBedrooms] = useState(String(initial?.bedrooms ?? 1));
  const [bathrooms, setBathrooms] = useState(String(initial?.bathrooms ?? 1));
  const [squareFeet, setSquareFeet] = useState(initial?.square_feet ? String(initial.square_feet) : '');
  const [rent, setRent] = useState(centsToInput(initial?.market_rent_cents ?? 0));
  const { submitting, error, errors, run } = useFormSubmit();

  function submit(e: FormEvent) {
    e.preventDefault();
    const local: Record<string, string> = {};
    const rentCents = rent.trim() === '' ? 0 : parseMoney(rent);
    if (rentCents === null) local.market_rent_cents = 'Enter an amount like 1450 or 1450.00';
    void run(
      () =>
        onSubmit({
          label,
          bedrooms: Number(bedrooms) || 0,
          bathrooms: Number(bathrooms) || 0,
          square_feet: squareFeet ? Number(squareFeet) : null,
          market_rent_cents: rentCents ?? 0,
        }),
      local,
    );
  }

  return (
    <form onSubmit={submit} noValidate>
      <FormError error={error} />
      <Field label="Unit" error={errors.label} hint="A number or name, e.g. 2B or Upstairs">
        {(id) => <input id={id} value={label} onChange={(e) => setLabel(e.target.value)} />}
      </Field>
      <div className="form-row">
        <Field label="Bedrooms" error={errors.bedrooms}>
          {(id) => (
            <input id={id} type="number" min={0} value={bedrooms} onChange={(e) => setBedrooms(e.target.value)} />
          )}
        </Field>
        <Field label="Bathrooms" error={errors.bathrooms}>
          {(id) => (
            <input
              id={id}
              type="number"
              min={0}
              step={0.5}
              value={bathrooms}
              onChange={(e) => setBathrooms(e.target.value)}
            />
          )}
        </Field>
        <Field label="Sq ft" error={errors.square_feet}>
          {(id) => (
            <input id={id} type="number" min={1} value={squareFeet} onChange={(e) => setSquareFeet(e.target.value)} />
          )}
        </Field>
      </div>
      <Field label="Market rent" error={errors.market_rent_cents} hint="What you would ask for it today">
        {(id) => <MoneyInput id={id} value={rent} onChange={(e) => setRent(e.target.value)} />}
      </Field>
      <div className="form-actions">
        <button type="button" className="button secondary" onClick={onCancel}>
          Cancel
        </button>
        <button type="submit" className="button" disabled={submitting}>
          {initial ? 'Save unit' : 'Add unit'}
        </button>
      </div>
    </form>
  );
}
