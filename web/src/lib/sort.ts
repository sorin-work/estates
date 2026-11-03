export type SortDir = 'asc' | 'desc';

export interface SortState<K extends string> {
  key: K | null;
  dir: SortDir;
}

/**
 * Returns a sorted copy. Strings compare ignoring case and accents, numbers
 * numerically; ties keep their original order.
 */
export function sortBy<T>(items: readonly T[], value: (item: T) => string | number, dir: SortDir): T[] {
  const collator = new Intl.Collator('en', { sensitivity: 'base', numeric: true });
  const sign = dir === 'asc' ? 1 : -1;
  return items
    .map((item, i) => ({ item, i, v: value(item) }))
    .sort((a, b) => {
      const c =
        typeof a.v === 'number' && typeof b.v === 'number' ? a.v - b.v : collator.compare(String(a.v), String(b.v));
      return c * sign || a.i - b.i;
    })
    .map((x) => x.item);
}

/** Clicking a column sorts by it; clicking it again flips the order. */
export function nextSort<K extends string>(current: SortState<K>, key: K, firstDir: SortDir = 'asc'): SortState<K> {
  if (current.key !== key) return { key, dir: firstDir };
  return { key, dir: current.dir === 'asc' ? 'desc' : 'asc' };
}
