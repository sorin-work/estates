const formatter = new Intl.NumberFormat('en-US', { style: 'currency', currency: 'USD' });
const compact = new Intl.NumberFormat('en-US', {
  style: 'currency',
  currency: 'USD',
  notation: 'compact',
  minimumFractionDigits: 0,
  maximumFractionDigits: 1,
});

/** Formats integer cents as dollars, e.g. 123456 -> "$1,234.56". */
export function formatMoney(cents: number): string {
  return formatter.format(cents / 100);
}

/** Short form for chart axes, e.g. 1250000 -> "$12.5K". */
export function formatMoneyCompact(cents: number): string {
  return compact.format(cents / 100);
}

/**
 * Parses what someone types into an amount field into cents. Accepts "1200",
 * "1,200.50" and " 12.05 ". Returns null for anything that is
 * not a valid non-negative amount with at most two decimals.
 */
export function parseMoney(input: string): number | null {
  const cleaned = input.trim().replace(/,/g, '');
  if (!/^\d+(\.\d{0,2})?$/.test(cleaned)) return null;
  const [whole = '0', fraction = ''] = cleaned.split('.');
  return Number(whole) * 100 + Number(fraction.padEnd(2, '0'));
}

/** Cents to the plain string shown in an input, e.g. 120050 -> "1200.50". */
export function centsToInput(cents: number): string {
  if (!cents) return '';
  return (cents / 100).toFixed(2);
}
