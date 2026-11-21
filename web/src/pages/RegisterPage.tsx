import { useQuery } from '@tanstack/react-query';
import { useState, type FormEvent } from 'react';
import { Link, useNavigate } from 'react-router';
import { authConfig } from '../api/auth';
import { useAuth } from '../auth/AuthContext';
import Field from '../components/Field';
import { FormError } from '../components/States';
import { useFormSubmit } from '../components/useFormSubmit';

export default function RegisterPage() {
  const { register } = useAuth();
  const navigate = useNavigate();
  const [name, setName] = useState('');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const { submitting, error, errors, run } = useFormSubmit();
  const config = useQuery({ queryKey: ['auth-config'], queryFn: authConfig, staleTime: Infinity });

  function submit(e: FormEvent) {
    e.preventDefault();
    const local: Record<string, string> = {};
    if (password.length > 0 && password.length < 8) local.password = 'Use at least 8 characters';
    void run(async () => {
      await register({ name, email, password });
      navigate('/', { replace: true });
    }, local);
  }

  if (config.data?.registration_open === false) {
    return (
      <div className="auth-page">
        <div className="auth-card">
          <div className="brand">
            <img src="/favicon.svg" alt="" width={32} height={32} />
            <span>Estates</span>
          </div>
          <h1>Sign-up is closed</h1>
          <p>This server does not take new accounts. Ask whoever runs it to create one for you.</p>
          <p className="muted center">
            Already have an account? <Link to="/login">Sign in</Link>
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className="auth-page">
      <form className="auth-card" onSubmit={submit} noValidate>
        <div className="brand">
          <img src="/favicon.svg" alt="" width={32} height={32} />
          <span>Estates</span>
        </div>
        <h1>Create your account</h1>
        <FormError error={error} />
        <Field label="Your name" error={errors.name}>
          {(id) => <input id={id} autoComplete="name" value={name} onChange={(e) => setName(e.target.value)} />}
        </Field>
        <Field label="Email" error={errors.email}>
          {(id) => (
            <input id={id} type="email" autoComplete="email" value={email} onChange={(e) => setEmail(e.target.value)} />
          )}
        </Field>
        <Field label="Password" error={errors.password} hint="At least 8 characters">
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
        <button type="submit" className="button block" disabled={submitting}>
          Create account
        </button>
        <p className="muted center">
          Already have an account? <Link to="/login">Sign in</Link>
        </p>
      </form>
    </div>
  );
}
