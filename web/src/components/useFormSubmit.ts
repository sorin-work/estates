import { useState } from 'react';
import { fieldErrors } from './States';

/**
 * Tracks the submitting state and server errors of a form. Local errors
 * (e.g. an amount that does not parse) are merged with the API's field
 * errors so a form only has one place to read them from.
 */
export function useFormSubmit() {
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<unknown>(null);
  const [localErrors, setLocalErrors] = useState<Record<string, string>>({});

  async function run(fn: () => Promise<unknown>, local: Record<string, string> = {}) {
    setLocalErrors(local);
    if (Object.keys(local).length > 0) {
      setError(null);
      return;
    }
    setSubmitting(true);
    setError(null);
    try {
      await fn();
    } catch (err) {
      setError(err);
    } finally {
      setSubmitting(false);
    }
  }

  return { submitting, error, errors: { ...fieldErrors(error), ...localErrors }, run };
}
