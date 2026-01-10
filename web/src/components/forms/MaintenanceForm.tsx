import { useQuery } from '@tanstack/react-query';
import { useState, type FormEvent } from 'react';
import { listProperties, listUnits } from '../../api/properties';
import type { MaintenanceInput, MaintenancePriority, MaintenanceRequest } from '../../api/types';
import { today } from '../../lib/dates';
import { maintenancePriorities, options } from '../../lib/labels';
import Field from '../Field';
import { FormError } from '../States';
import { useFormSubmit } from '../useFormSubmit';

export default function MaintenanceForm({
  initial,
  propertyId,
  onSubmit,
  onCancel,
}: {
  initial?: MaintenanceRequest;
  propertyId?: string;
  onSubmit: (input: MaintenanceInput) => Promise<unknown>;
  onCancel: () => void;
}) {
  const [form, setForm] = useState<MaintenanceInput>({
    property_id: initial?.property_id ?? propertyId ?? '',
    unit_id: initial?.unit_id ?? null,
    title: initial?.title ?? '',
    description: initial?.description ?? '',
    priority: initial?.priority ?? 'normal',
    vendor: initial?.vendor ?? '',
    reported_on: initial?.reported_on ?? today(),
  });
  const properties = useQuery({ queryKey: ['properties'], queryFn: listProperties });
  const units = useQuery({
    queryKey: ['units', form.property_id],
    queryFn: () => listUnits(form.property_id),
    enabled: form.property_id !== '',
  });
  const { submitting, error, errors, run } = useFormSubmit();

  function submit(e: FormEvent) {
    e.preventDefault();
    void run(() => onSubmit(form));
  }

  return (
    <form onSubmit={submit} noValidate>
      <FormError error={error} />
      <Field label="What needs doing?" error={errors.title}>
        {(id) => (
          <input
            id={id}
            value={form.title}
            placeholder="e.g. Kitchen tap dripping"
            onChange={(e) => setForm({ ...form, title: e.target.value })}
          />
        )}
      </Field>
      <div className="form-row">
        <Field label="Property" error={errors.property_id}>
          {(id) => (
            <select
              id={id}
              value={form.property_id}
              onChange={(e) => setForm({ ...form, property_id: e.target.value, unit_id: null })}
            >
              <option value="">Choose a property…</option>
              {properties.data?.map((p) => (
                <option key={p.id} value={p.id}>
                  {p.name}
                </option>
              ))}
            </select>
          )}
        </Field>
        <Field label="Unit" error={errors.unit_id}>
          {(id) => (
            <select
              id={id}
              value={form.unit_id ?? ''}
              disabled={!form.property_id}
              onChange={(e) => setForm({ ...form, unit_id: e.target.value || null })}
            >
              <option value="">Whole property / common area</option>
              {units.data?.map((u) => (
                <option key={u.id} value={u.id}>
                  {u.label}
                </option>
              ))}
            </select>
          )}
        </Field>
      </div>
      <div className="form-row">
        <Field label="Priority" error={errors.priority}>
          {(id) => (
            <select
              id={id}
              value={form.priority}
              onChange={(e) => setForm({ ...form, priority: e.target.value as MaintenancePriority })}
            >
              {options(maintenancePriorities).map((o) => (
                <option key={o.value} value={o.value}>
                  {o.label}
                </option>
              ))}
            </select>
          )}
        </Field>
        <Field label="Reported on" error={errors.reported_on}>
          {(id) => (
            <input
              id={id}
              type="date"
              value={form.reported_on}
              onChange={(e) => setForm({ ...form, reported_on: e.target.value })}
            />
          )}
        </Field>
      </div>
      <Field label="Vendor" error={errors.vendor} hint="Who is doing the work, if known">
        {(id) => <input id={id} value={form.vendor} onChange={(e) => setForm({ ...form, vendor: e.target.value })} />}
      </Field>
      <Field label="Details" error={errors.description}>
        {(id) => (
          <textarea
            id={id}
            rows={3}
            value={form.description}
            onChange={(e) => setForm({ ...form, description: e.target.value })}
          />
        )}
      </Field>
      <div className="form-actions">
        <button type="button" className="button secondary" onClick={onCancel}>
          Cancel
        </button>
        <button type="submit" className="button" disabled={submitting}>
          {initial ? 'Save request' : 'Log request'}
        </button>
      </div>
    </form>
  );
}
