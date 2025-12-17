import type { ReactNode } from 'react';
import { ApiError } from '../api/client';

export function Loading({ label = 'Loading…' }: { label?: string }) {
  return (
    <div className="state" role="status">
      {label}
    </div>
  );
}

export function ErrorState({ error }: { error: unknown }) {
  const message = error instanceof Error ? error.message : 'Something went wrong.';
  const notFound = error instanceof ApiError && error.status === 404;
  return (
    <div className="state state-error" role="alert">
      {notFound ? 'This record does not exist or was deleted.' : message}
    </div>
  );
}

export function EmptyState({ title, children }: { title: string; children?: ReactNode }) {
  return (
    <div className="empty">
      <h3>{title}</h3>
      {children && <div className="muted">{children}</div>}
    </div>
  );
}

/** Shows the API error message for a failed form submit. */
export function FormError({ error }: { error: unknown }) {
  if (!error) return null;
  let message = 'Something went wrong.';
  if (error instanceof ApiError) {
    message = error.code === 'validation_failed' ? 'Please fix the highlighted fields.' : error.message;
  } else if (error instanceof Error) {
    message = error.message;
  }
  return (
    <div className="form-error" role="alert">
      {message}
    </div>
  );
}

export function fieldErrors(error: unknown): Record<string, string> {
  return error instanceof ApiError ? error.fields : {};
}
