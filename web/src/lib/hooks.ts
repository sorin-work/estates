import { useCallback, useEffect, useState } from 'react';
import { useSearchParams } from 'react-router';

/** Returns value once it has stopped changing for ms milliseconds. */
export function useDebounced<T>(value: T, ms: number): T {
  const [debounced, setDebounced] = useState(value);
  useEffect(() => {
    const t = setTimeout(() => setDebounced(value), ms);
    return () => clearTimeout(t);
  }, [value, ms]);
  return debounced;
}

/**
 * The page's search params, with a setter for one filter that drops it when
 * empty and replaces the history entry, so filtering does not fill up the
 * back button.
 */
export function useFilterParams() {
  const [params, setParams] = useSearchParams();
  const setFilter = useCallback(
    (key: string, value: string) => {
      const next = new URLSearchParams(params);
      if (value) next.set(key, value);
      else next.delete(key);
      setParams(next, { replace: true });
    },
    [params, setParams],
  );
  return [params, setFilter, setParams] as const;
}
