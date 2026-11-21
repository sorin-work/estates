/** The browser's time zone, such as "Europe/Bucharest", or "" if unknown. */
export function browserTimeZone(): string {
  try {
    return Intl.DateTimeFormat().resolvedOptions().timeZone ?? '';
  } catch {
    return '';
  }
}

const fallback = ['UTC', 'Europe/London', 'Europe/Berlin', 'America/New_York', 'America/Los_Angeles'];

/**
 * Every time zone the browser knows, sorted, always including UTC and the
 * ones passed in (the saved zone and the browser's), so the select can show
 * them even where the list is short or missing.
 */
export function timeZoneOptions(...include: string[]): string[] {
  let all: string[];
  try {
    all = Intl.supportedValuesOf('timeZone');
  } catch {
    all = fallback;
  }
  const set = new Set(all);
  for (const z of ['UTC', ...include]) if (z) set.add(z);
  return [...set].sort((a, b) => (a === 'UTC' ? -1 : b === 'UTC' ? 1 : a.localeCompare(b)));
}

/** "America/Los_Angeles" as "America / Los Angeles". */
export function timeZoneLabel(zone: string): string {
  return zone.replaceAll('_', ' ').replaceAll('/', ' / ');
}
