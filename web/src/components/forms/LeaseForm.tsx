import { useQuery } from '@tanstack/react-query';
import { useState, type FormEvent } from 'react';
import { listUnits } from '../../api/properties';
import { listTenants } from '../../api/tenants';
import type { Lease, LeaseInput } from '../../api/types';
import { addDays, addMonths, today } from '../../lib/dates';
import { centsToInput, parseMoney } from '../../lib/money';
import Field from '../Field';
import { FormError } from '../States';
import { useFormSubmit } from '../useFormSubmit';
import MoneyInput from './MoneyInput';

export interface LeaseFormDefaults {
  unit_id?: string;
  tenant_id?: string;
}

export default function LeaseForm({
  initial,
  defaults,
  onSubmit,
  onCancel,
}: {
  initial?: Lease;
  defaults?: LeaseFormDefaults;
  onSubmit: (input: LeaseInput) => Promise<unknown>;
  onCancel: () => void;
}) {
  const units = useQuery({ queryKey: ['units'], queryFn: () => listUnits() });
  const tenants = useQuery({ queryKey: ['tenants', ''], queryFn: () => listTenants() });

  const start = initial?.start_date ?? today();
  const [unitId, setUnitId] = useState(initial?.unit_id ?? defaults?.unit_id ?? '');
  const [tenantId, setTenantId] = useState(initial?.tenant_id ?? defaults?.tenant_id ?? '');
  const [startDate, setStartDate] = useState(start);
  const [endDate, setEndDate] = useState(initial?.end_date ?? addDays(addMonths(start, 12), -1));
  const [rent, setRent] = useState(centsToInput(initial?.rent_cents ?? 0));
  const [deposit, setDeposit] = useState(centsToInput(initial?.deposit_cents ?? 0));
  const [dueDay, setDueDay] = useState(String(initial?.due_day ?? 1));
  const [graceDays, setGraceDays] = useState(String(initial?.grace_days ?? 5));
  const [lateFee, setLateFee] = useState(centsToInput(initial?.late_fee_cents ?? 0));
  const [notes, setNotes] = useState(initial?.notes ?? '');
  const { submitting, error, errors, run } = useFormSubmit();

  function changeStart(value: string) {
    setStartDate(value);
    if (!initial && value) setEndDate(addDays(addMonths(value, 12), -1));
  }

  function submit(e: FormEvent) {
    e.preventDefault();
    const local: Record<string, string> = {};
    const rentCents = parseMoney(rent);
    const depositCents = deposit.trim() === '' ? 0 : parseMoney(deposit);
    const lateFeeCents = lateFee.trim() === '' ? 0 : parseMoney(lateFee);
    if (rentCents === null || rentCents === 0) local.rent_cents = 'Enter the monthly rent';
    if (depositCents === null) local.deposit_cents = 'Enter an amount like 1500.00';
    if (lateFeeCents === null) local.late_fee_cents = 'Enter an amount like 50.00';

    void run(
      () =>
        onSubmit({
          unit_id: unitId,
          tenant_id: tenantId,
          start_date: startDate,
          end_date: endDate,
          rent_cents: rentCents ?? 0,
          deposit_cents: depositCents ?? 0,
          due_day: Number(dueDay) || 1,
          grace_days: Number(graceDays) || 0,
          late_fee_cents: lateFeeCents ?? 0,
          notes,
        }),
      local,
    );
  }

  return (
    <form onSubmit={submit} noValidate>
      <FormError error={error} />
      <div className="form-row">
        <Field label="Unit" error={errors.unit_id}>
          {(id) => (
            <select id={id} value={unitId} onChange={(e) => setUnitId(e.target.value)}>
              <option value="">Choose a unit…</option>
              {units.data?.map((u) => (
                <option key={u.id} value={u.id}>
                  {u.property_name} · {u.label}
                  {u.current_tenant && u.id !== initial?.unit_id ? ` (let to ${u.current_tenant})` : ''}
                </option>
              ))}
            </select>
          )}
        </Field>
        <Field label="Tenant" error={errors.tenant_id}>
          {(id) => (
            <select id={id} value={tenantId} onChange={(e) => setTenantId(e.target.value)}>
              <option value="">Choose a tenant…</option>
              {tenants.data?.map((t) => (
                <option key={t.id} value={t.id}>
                  {t.first_name} {t.last_name}
                </option>
              ))}
            </select>
          )}
        </Field>
      </div>
      <div className="form-row">
        <Field label="Start date" error={errors.start_date}>
          {(id) => <input id={id} type="date" value={startDate} onChange={(e) => changeStart(e.target.value)} />}
        </Field>
        <Field label="End date" error={errors.end_date}>
          {(id) => <input id={id} type="date" value={endDate} onChange={(e) => setEndDate(e.target.value)} />}
        </Field>
      </div>
      <div className="form-row">
        <Field label="Monthly rent" error={errors.rent_cents}>
          {(id) => <MoneyInput id={id} value={rent} onChange={(e) => setRent(e.target.value)} />}
        </Field>
        <Field label="Security deposit" error={errors.deposit_cents}>
          {(id) => <MoneyInput id={id} value={deposit} onChange={(e) => setDeposit(e.target.value)} />}
        </Field>
      </div>
      <div className="form-row">
        <Field label="Rent due on day" error={errors.due_day} hint="31 means the last day of the month">
          {(id) => (
            <input id={id} type="number" min={1} max={31} value={dueDay} onChange={(e) => setDueDay(e.target.value)} />
          )}
        </Field>
        <Field label="Grace period (days)" error={errors.grace_days}>
          {(id) => (
            <input
              id={id}
              type="number"
              min={0}
              max={31}
              value={graceDays}
              onChange={(e) => setGraceDays(e.target.value)}
            />
          )}
        </Field>
        <Field label="Late fee" error={errors.late_fee_cents} hint="Added once per late month">
          {(id) => <MoneyInput id={id} value={lateFee} onChange={(e) => setLateFee(e.target.value)} />}
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
          {initial ? 'Save lease' : 'Create lease'}
        </button>
      </div>
    </form>
  );
}
