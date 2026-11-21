import { browserTimeZone, timeZoneLabel } from './timezones';

let accountZone = '';

/**
 * Sets the time zone "today" is worked out in. The API decides what today
 * is from the account's saved zone, so the forms must use the same one, not
 * the browser's.
 */
export function setAccountTimeZone(zone: string): void {
  accountZone = zone;
}

/** Today's date as YYYY-MM-DD in the account's time zone, or the browser's. */
export function today(): string {
  if (accountZone) {
    try {
      return isoDateIn(new Date(), accountZone);
    } catch {
      // An unknown zone: fall back to the browser's.
    }
  }
  return toISODate(new Date());
}

/** The date of d in the given IANA time zone, as YYYY-MM-DD. */
export function isoDateIn(d: Date, zone: string): string {
  const parts = new Intl.DateTimeFormat('en-US', {
    timeZone: zone,
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
  }).formatToParts(d);
  const get = (type: string) => parts.find((p) => p.type === type)?.value ?? '';
  return `${get('year')}-${get('month')}-${get('day')}`;
}

/**
 * A hint for date fields when this device is in another time zone than the
 * account, where "today" can be a different day.
 */
export function accountZoneHint(): string | undefined {
  const device = browserTimeZone();
  if (!accountZone || !device || device === accountZone) return undefined;
  return `Today is ${formatDate(today())} in ${timeZoneLabel(accountZone)}, your account's time zone.`;
}

export function toISODate(d: Date): string {
  const y = d.getFullYear();
  const m = String(d.getMonth() + 1).padStart(2, '0');
  const day = String(d.getDate()).padStart(2, '0');
  return `${y}-${m}-${day}`;
}

/** Parses YYYY-MM-DD as a local date (not UTC, which can shift the day). */
export function parseISODate(s: string): Date {
  const [y, m, d] = s.split('-').map(Number);
  return new Date(y ?? 1970, (m ?? 1) - 1, d ?? 1);
}

const dateFormat = new Intl.DateTimeFormat('en-US', { month: 'short', day: 'numeric', year: 'numeric' });
const monthFormat = new Intl.DateTimeFormat('en-US', { month: 'short' });
const monthYearFormat = new Intl.DateTimeFormat('en-US', { month: 'long', year: 'numeric' });

/** "2026-03-05" -> "Mar 5, 2026". Empty and null values become an em dash. */
export function formatDate(s: string | null | undefined): string {
  if (!s) return '—';
  return dateFormat.format(parseISODate(s));
}

/** "2026-03" -> "Mar". */
export function formatMonthShort(month: string): string {
  return monthFormat.format(parseISODate(`${month}-01`));
}

/** "2026-03" -> "March 2026". */
export function formatMonthLong(month: string): string {
  return monthYearFormat.format(parseISODate(`${month}-01`));
}

export function addDays(s: string, days: number): string {
  const d = parseISODate(s);
  d.setDate(d.getDate() + days);
  return toISODate(d);
}

/** Adds months, keeping the day where possible (Jan 31 + 1 = Feb 28). */
export function addMonths(s: string, months: number): string {
  const d = parseISODate(s);
  const day = d.getDate();
  d.setDate(1);
  d.setMonth(d.getMonth() + months);
  const lastDay = new Date(d.getFullYear(), d.getMonth() + 1, 0).getDate();
  d.setDate(Math.min(day, lastDay));
  return toISODate(d);
}

/** Whole days from a to b (negative if b is earlier). */
export function daysBetween(a: string, b: string): number {
  const ms = parseISODate(b).getTime() - parseISODate(a).getTime();
  return Math.round(ms / 86_400_000);
}

export function monthRange(month: string): { from: string; to: string } {
  const from = `${month}-01`;
  return { from, to: addDays(addMonths(from, 1), -1) };
}

/** "in 12 days", "today", "3 days ago". */
export function relativeDays(days: number): string {
  if (days === 0) return 'today';
  if (days === 1) return 'tomorrow';
  if (days === -1) return 'yesterday';
  return days > 0 ? `in ${days} days` : `${-days} days ago`;
}
