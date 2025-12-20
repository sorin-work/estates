import { useState, type FormEvent } from 'react';
import type { Property, PropertyInput, PropertyKind } from '../../api/types';
import { options, propertyKinds } from '../../lib/labels';
import Field from '../Field';
import { FormError } from '../States';
import { useFormSubmit } from '../useFormSubmit';

export default function PropertyForm({
  initial,
  onSubmit,
  onCancel,
}: {
  initial?: Property;
  onSubmit: (input: PropertyInput) => Promise<unknown>;
  onCancel: () => void;
}) {
  const [form, setForm] = useState<PropertyInput>({
    name: initial?.name ?? '',
    kind: initial?.kind ?? 'single_family',
    address_line1: initial?.address_line1 ?? '',
    address_line2: initial?.address_line2 ?? '',
    city: initial?.city ?? '',
    region: initial?.region ?? '',
    postal_code: initial?.postal_code ?? '',
    notes: initial?.notes ?? '',
  });
  const { submitting, error, errors, run } = useFormSubmit();
  const set = (key: keyof PropertyInput) => (e: { target: { value: string } }) =>
    setForm({ ...form, [key]: e.target.value });

  function submit(e: FormEvent) {
    e.preventDefault();
    void run(() => onSubmit(form));
  }

  return (
    <form onSubmit={submit} noValidate>
      <FormError error={error} />
      <div className="form-grid">
        <Field label="Name" error={errors.name}>
          {(id) => <input id={id} value={form.name} onChange={set('name')} placeholder="e.g. Maple Street Duplex" />}
        </Field>
        <Field label="Type" error={errors.kind}>
          {(id) => (
            <select
              id={id}
              value={form.kind}
              onChange={(e) => setForm({ ...form, kind: e.target.value as PropertyKind })}
            >
              {options(propertyKinds).map((o) => (
                <option key={o.value} value={o.value}>
                  {o.label}
                </option>
              ))}
            </select>
          )}
        </Field>
        <Field label="Address" error={errors.address_line1}>
          {(id) => <input id={id} value={form.address_line1} onChange={set('address_line1')} />}
        </Field>
        <Field label="Address line 2" error={errors.address_line2}>
          {(id) => <input id={id} value={form.address_line2} onChange={set('address_line2')} />}
        </Field>
        <Field label="City" error={errors.city}>
          {(id) => <input id={id} value={form.city} onChange={set('city')} />}
        </Field>
        <div className="form-row">
          <Field label="State / region" error={errors.region}>
            {(id) => <input id={id} value={form.region} onChange={set('region')} />}
          </Field>
          <Field label="Postal code" error={errors.postal_code}>
            {(id) => <input id={id} value={form.postal_code} onChange={set('postal_code')} />}
          </Field>
        </div>
      </div>
      <Field label="Notes" error={errors.notes}>
        {(id) => <textarea id={id} rows={3} value={form.notes} onChange={set('notes')} />}
      </Field>
      <div className="form-actions">
        <button type="button" className="button secondary" onClick={onCancel}>
          Cancel
        </button>
        <button type="submit" className="button" disabled={submitting}>
          {initial ? 'Save changes' : 'Add property'}
        </button>
      </div>
    </form>
  );
}
