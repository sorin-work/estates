// Amounts are always integer cents (or the minor unit of the currency). The
// currency only changes how they are shown; it is set once the signed in
// user is known.
let currency = 'USD';
let formatter = full(currency);
let compact = short(currency);

function full(code: string) {
  return new Intl.NumberFormat('en-US', { style: 'currency', currency: code });
}

function short(code: string) {
  return new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: code,
    notation: 'compact',
    minimumFractionDigits: 0,
    maximumFractionDigits: 1,
  });
}

/** Switches the currency used by formatMoney and friends. */
export function setCurrency(code: string): void {
  if (code === currency) return;
  try {
    formatter = full(code);
    compact = short(code);
    currency = code;
  } catch {
    // An unknown code would throw a RangeError; keep the previous one.
  }
}

export function getCurrency(): string {
  return currency;
}

/** The symbol shown next to amount fields, e.g. "$", "€" or "CHF". */
export function currencySymbol(): string {
  return formatter.formatToParts(0).find((p) => p.type === 'currency')?.value ?? currency;
}

/** Formats integer cents, e.g. 123456 -> "$1,234.56". */
export function formatMoney(cents: number): string {
  return formatter.format(cents / 100);
}

/** Short form for chart axes, e.g. 1250000 -> "$12.5K". */
export function formatMoneyCompact(cents: number): string {
  return compact.format(cents / 100);
}

/**
 * Parses what someone types into an amount field into cents. Accepts "1200",
 * "1,200.50", "$1,200.5", "€ 80" and " 12.05 ". Returns null for anything
 * that is not a valid non-negative amount with at most two decimals.
 */
export function parseMoney(input: string): number | null {
  const cleaned = input
    .trim()
    .replace(/^(?:[A-Z]{0,2}\$|[€£]|[A-Z]{3})\s*/i, '')
    .replace(/,/g, '');
  if (!/^\d+(\.\d{0,2})?$/.test(cleaned)) return null;
  const [whole = '0', fraction = ''] = cleaned.split('.');
  return Number(whole) * 100 + Number(fraction.padEnd(2, '0'));
}

/** Cents to the plain string shown in an input, e.g. 120050 -> "1200.50". */
export function centsToInput(cents: number): string {
  if (!cents) return '';
  return (cents / 100).toFixed(2);
}
