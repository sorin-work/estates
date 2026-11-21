import { useEffect, useState, type FormEvent } from 'react';
import { Link, useLocation, useNavigate } from 'react-router';
import { resetPassword } from '../api/auth';
import { setToken } from '../api/client';
import { useAuth } from '../auth/AuthContext';
import Field from '../components/Field';
import { FormError } from '../components/States';
import { useFormSubmit } from '../components/useFormSubmit';

/** The token is after # in the emailed link, so it never reaches a server log. */
function tokenFrom(hash: string): string {
  return new URLSearchParams(hash.replace(/^#/, '')).get('token') ?? '';
}

export default function ResetPasswordPage() {
  const location = useLocation();
  const navigate = useNavigate();
  const { updateUser } = useAuth();
  const [token] = useState(() => tokenFrom(location.hash));
  // Keep the token out of the address bar and the browser history.
  useEffect(() => {
    if (window.location.hash) window.history.replaceState(window.history.state, '', window.location.pathname);
  }, []);
  const [password, setPassword] = useState('');
  const [confirm, setConfirm] = useState('');
  const { submitting, error, errors, run } = useFormSubmit();

  function submit(e: FormEvent) {
    e.preventDefault();
    const local: Record<string, string> = {};
    if (password !== confirm) local.confirm = 'does not match the new password';
    void run(async () => {
      const res = await resetPassword({ token, new_password: password });
      setToken(res.token);
      updateUser(res.user);
      navigate('/', { replace: true });
    }, local);
  }

  return (
    <div className="auth-page">
      <form className="auth-card" onSubmit={submit} noValidate>
        <div className="brand">
          <img src="/favicon.svg" alt="" width={32} height={32} />
          <span>Estates</span>
        </div>
        <h1>Choose a new password</h1>
        {!token ? (
          <p role="alert">
            This link is not complete. Open the link from the email again, or{' '}
            <Link to="/forgot-password">ask for a new one</Link>.
          </p>
        ) : (
          <>
            <p className="muted">Every device signed in to this account will be signed out.</p>
            <FormError error={error} />
            <Field label="New password" error={errors.new_password} hint="At least 8 characters.">
              {(id) => (
                <input
                  id={id}
                  type="password"
                  autoComplete="new-password"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                />
              )}
            </Field>
            <Field label="Repeat new password" error={errors.confirm}>
              {(id) => (
                <input
                  id={id}
                  type="password"
                  autoComplete="new-password"
                  value={confirm}
                  onChange={(e) => setConfirm(e.target.value)}
                />
              )}
            </Field>
            <button type="submit" className="button block" disabled={submitting}>
              Set the new password
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
