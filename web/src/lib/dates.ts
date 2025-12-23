/** Today's date as YYYY-MM-DD in the browser's timezone. */
export function today(): string {
  return toISODate(new Date());
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
