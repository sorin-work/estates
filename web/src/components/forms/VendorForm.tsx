import { useState, type FormEvent } from 'react';
import type { Vendor, VendorInput, VendorTrade } from '../../api/types';
import { options, vendorTrades } from '../../lib/labels';
import Field from '../Field';
import { FormError } from '../States';
import { useFormSubmit } from '../useFormSubmit';

export default function VendorForm({
  initial,
  onSubmit,
  onCancel,
}: {
  initial?: Vendor;
  onSubmit: (input: VendorInput) => Promise<unknown>;
  onCancel: () => void;
}) {
  const [form, setForm] = useState<VendorInput>({
    name: initial?.name ?? '',
    trade: initial?.trade ?? 'handyman',
    contact_name: initial?.contact_name ?? '',
    phone: initial?.phone ?? '',
    email: initial?.email ?? '',
    notes: initial?.notes ?? '',
  });
  const { submitting, error, errors, run } = useFormSubmit();
  const set = (key: keyof VendorInput) => (e: { target: { value: string } }) =>
    setForm({ ...form, [key]: e.target.value });

  function submit(e: FormEvent) {
    e.preventDefault();
    void run(() => onSubmit(form));
  }

  return (
    <form onSubmit={submit} noValidate>
      <FormError error={error} />
      <div className="form-row">
        <Field label="Name" error={errors.name}>
          {(id) => <input id={id} value={form.name} onChange={set('name')} />}
        </Field>
        <Field label="Trade" error={errors.trade}>
          {(id) => (
            <select
              id={id}
              value={form.trade}
              onChange={(e) => setForm({ ...form, trade: e.target.value as VendorTrade })}
            >
              {options(vendorTrades).map((o) => (
                <option key={o.value} value={o.value}>
                  {o.label}
                </option>
              ))}
            </select>
          )}
        </Field>
      </div>
      <Field label="Contact" error={errors.contact_name} hint="The person you usually deal with">
        {(id) => <input id={id} value={form.contact_name} onChange={set('contact_name')} />}
      </Field>
      <div className="form-row">
        <Field label="Phone" error={errors.phone}>
          {(id) => <input id={id} type="tel" value={form.phone} onChange={set('phone')} />}
        </Field>
        <Field label="Email" error={errors.email}>
          {(id) => <input id={id} type="email" value={form.email} onChange={set('email')} />}
        </Field>
      </div>
      <Field label="Notes" error={errors.notes}>
        {(id) => <textarea id={id} rows={3} value={form.notes} onChange={set('notes')} />}
      </Field>
      <div className="form-actions">
        <button type="button" className="button secondary" onClick={onCancel}>
          Cancel
        </button>
        <button type="submit" className="button" disabled={submitting}>
          {initial ? 'Save vendor' : 'Add vendor'}
        </button>
      </div>
    </form>
  );
}
