import { useState, type FormEvent } from 'react';
import { changePassword, updateMe } from '../api/auth';
import { useAuth } from '../auth/AuthContext';
import Field from '../components/Field';
import PageHeader from '../components/PageHeader';
import { FormError } from '../components/States';
import { useFormSubmit } from '../components/useFormSubmit';
import { currencies } from '../lib/labels';

function ProfileForm() {
  const { user, updateUser } = useAuth();
  const [name, setName] = useState(user?.name ?? '');
  const [currency, setCurrency] = useState(user?.currency ?? 'USD');
  const [returnDays, setReturnDays] = useState(String(user?.deposit_return_days ?? 30));
  const [saved, setSaved] = useState(false);
  const { submitting, error, errors, run } = useFormSubmit();

  function submit(e: FormEvent) {
    e.preventDefault();
    setSaved(false);
    void run(async () => {
      updateUser(await updateMe({ name, currency, deposit_return_days: Number(returnDays) || 0 }));
      setSaved(true);
    });
  }

  return (
    <form onSubmit={submit} noValidate>
      <FormError error={error} />
      <Field label="Email">{(id) => <input id={id} value={user?.email ?? ''} disabled />}</Field>
      <Field label="Name" error={errors.name}>
        {(id) => <input id={id} value={name} onChange={(e) => setName(e.target.value)} />}
      </Field>
      <Field label="Currency" error={errors.currency} hint="Only changes how amounts are shown.">
        {(id) => (
          <select id={id} value={currency} onChange={(e) => setCurrency(e.target.value)}>
            {Object.entries(currencies).map(([code, label]) => (
              <option key={code} value={code}>
                {label} ({code})
              </option>
            ))}
          </select>
        )}
      </Field>
      <Field
        label="Deposit return deadline"
        error={errors.deposit_return_days}
        hint="Days after move-out to return a deposit. Check your state's rules."
      >
        {(id) => (
          <input
            id={id}
            type="number"
            min={1}
            max={90}
            value={returnDays}
            onChange={(e) => setReturnDays(e.target.value)}
          />
        )}
      </Field>
      <div className="form-actions">
        {saved && (
          <span className="muted" role="status">
            Saved
          </span>
        )}
        <button type="submit" className="button" disabled={submitting}>
          Save
        </button>
      </div>
    </form>
  );
}

function PasswordForm() {
  const empty = { current: '', next: '', confirm: '' };
  const [form, setForm] = useState(empty);
  const [saved, setSaved] = useState(false);
  const { submitting, error, errors, run } = useFormSubmit();

  function submit(e: FormEvent) {
    e.preventDefault();
    setSaved(false);
    const local: Record<string, string> = {};
    if (form.next !== form.confirm) local.confirm = 'does not match the new password';
    void run(async () => {
      await changePassword({ current_password: form.current, new_password: form.next });
      setForm(empty);
      setSaved(true);
    }, local);
  }

  const set = (key: keyof typeof empty) => (e: { target: { value: string } }) =>
    setForm({ ...form, [key]: e.target.value });

  return (
    <form onSubmit={submit} noValidate>
      <FormError error={error} />
      <Field label="Current password" error={errors.current_password}>
        {(id) => (
          <input
            id={id}
            type="password"
            autoComplete="current-password"
            value={form.current}
            onChange={set('current')}
          />
        )}
      </Field>
      <div className="form-row">
        <Field label="New password" error={errors.new_password} hint="At least 8 characters.">
          {(id) => (
            <input id={id} type="password" autoComplete="new-password" value={form.next} onChange={set('next')} />
          )}
        </Field>
        <Field label="Repeat new password" error={errors.confirm}>
          {(id) => (
            <input id={id} type="password" autoComplete="new-password" value={form.confirm} onChange={set('confirm')} />
          )}
        </Field>
      </div>
      <div className="form-actions">
        {saved && (
          <span className="muted" role="status">
            Password changed
          </span>
        )}
        <button type="submit" className="button" disabled={submitting}>
          Change password
        </button>
      </div>
    </form>
  );
}

export default function SettingsPage() {
  return (
    <>
      <PageHeader title="Settings" />
      <div className="settings">
        <section className="card">
          <div className="card-header">
            <h2>Account</h2>
          </div>
          <ProfileForm />
        </section>
        <section className="card">
          <div className="card-header">
            <h2>Password</h2>
          </div>
          <PasswordForm />
        </section>
      </div>
    </>
  );
}
