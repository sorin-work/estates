import { useState, type FormEvent } from 'react';
import type { Tenant, TenantInput } from '../../api/types';
import Field from '../Field';
import { FormError } from '../States';
import { useFormSubmit } from '../useFormSubmit';

export default function TenantForm({
  initial,
  onSubmit,
  onCancel,
}: {
  initial?: Tenant;
  onSubmit: (input: TenantInput) => Promise<unknown>;
  onCancel: () => void;
}) {
  const [form, setForm] = useState<TenantInput>({
    first_name: initial?.first_name ?? '',
    last_name: initial?.last_name ?? '',
    email: initial?.email ?? '',
    phone: initial?.phone ?? '',
    notes: initial?.notes ?? '',
  });
  const { submitting, error, errors, run } = useFormSubmit();
  const set = (key: keyof TenantInput) => (e: { target: { value: string } }) =>
    setForm({ ...form, [key]: e.target.value });

  function submit(e: FormEvent) {
    e.preventDefault();
    void run(() => onSubmit(form));
  }

  return (
    <form onSubmit={submit} noValidate>
      <FormError error={error} />
      <div className="form-row">
        <Field label="First name" error={errors.first_name}>
          {(id) => <input id={id} value={form.first_name} onChange={set('first_name')} />}
        </Field>
        <Field label="Last name" error={errors.last_name}>
          {(id) => <input id={id} value={form.last_name} onChange={set('last_name')} />}
        </Field>
      </div>
      <div className="form-row">
        <Field label="Email" error={errors.email}>
          {(id) => <input id={id} type="email" value={form.email} onChange={set('email')} />}
        </Field>
        <Field label="Phone" error={errors.phone}>
          {(id) => <input id={id} type="tel" value={form.phone} onChange={set('phone')} />}
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
          {initial ? 'Save tenant' : 'Add tenant'}
        </button>
      </div>
    </form>
  );
}
