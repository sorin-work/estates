import { useState, type FormEvent } from 'react';
import { Link } from 'react-router';
import { forgotPassword } from '../api/auth';
import Field from '../components/Field';
import { FormError } from '../components/States';
import { useFormSubmit } from '../components/useFormSubmit';

export default function ForgotPasswordPage() {
  const [email, setEmail] = useState('');
  const [sent, setSent] = useState('');
  const { submitting, error, errors, run } = useFormSubmit();

  function submit(e: FormEvent) {
    e.preventDefault();
    void run(async () => {
      const res = await forgotPassword(email);
      setSent(res.message);
    });
  }

  return (
    <div className="auth-page">
      <form className="auth-card" onSubmit={submit} noValidate>
        <div className="brand">
          <img src="/favicon.svg" alt="" width={32} height={32} />
          <span>Estates</span>
        </div>
        <h1>Reset your password</h1>
        {sent ? (
          <p role="status">{sent} The link works for 30 minutes.</p>
        ) : (
          <>
            <p className="muted">We will email you a link to choose a new password.</p>
            <FormError error={error} />
            <Field label="Email" error={errors.email}>
              {(id) => (
                <input
                  id={id}
                  type="email"
                  autoComplete="email"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                />
              )}
            </Field>
            <button type="submit" className="button block" disabled={submitting}>
              Send the link
            </button>
          </>
        )}
        <p className="muted center">
          <Link to="/login">Back to sign in</Link>
        </p>
      </form>
    </div>
  );
}
