import { afterEach, describe, expect, it } from 'vitest';
import { centsToInput, currencySymbol, formatMoney, formatMoneyCompact, parseMoney, setCurrency } from './money';

afterEach(() => setCurrency('USD'));

describe('formatMoney', () => {
  it('formats cents as dollars', () => {
    expect(formatMoney(123456)).toBe('$1,234.56');
    expect(formatMoney(5)).toBe('$0.05');
    expect(formatMoney(-2500)).toBe('-$25.00');
  });

  it('has a compact form for axes', () => {
    expect(formatMoneyCompact(1250000)).toBe('$12.5K');
    expect(formatMoneyCompact(0)).toBe('$0');
  });

  it('follows the selected currency', () => {
    setCurrency('EUR');
    expect(formatMoney(123456)).toBe('€1,234.56');
    expect(currencySymbol()).toBe('€');
    setCurrency('CHF');
    expect(formatMoney(5000)).toBe('CHF\u00a050.00');
  });

  it('ignores an unknown currency', () => {
    setCurrency('NOPE');
    expect(formatMoney(100)).toBe('$1.00');
  });
});

describe('parseMoney', () => {
  it.each([
    ['1200', 120000],
    ['1,200.50', 120050],
    ['$1,200.5', 120050],
    [' 12.05 ', 1205],
    ['0', 0],
    ['7.', 700],
    ['€ 80', 8000],
    ['CA$15', 1500],
    ['CHF 12.50', 1250],
  ])('parses %j as %d cents', (input, cents) => {
    expect(parseMoney(input)).toBe(cents);
  });

  it.each(['', 'abc', '-5', '1.234', '1.2.3', '$'])('rejects %j', (input) => {
    expect(parseMoney(input)).toBeNull();
  });

  it('never goes through floating point', () => {
    // 0.29 * 100 is 28.999999999999996 in floating point.
    expect(parseMoney('0.29')).toBe(29);
    expect(parseMoney('1234567.89')).toBe(123456789);
  });
});

describe('centsToInput', () => {
  it('shows two decimals and leaves zero empty', () => {
    expect(centsToInput(145000)).toBe('1450.00');
    expect(centsToInput(5)).toBe('0.05');
    expect(centsToInput(0)).toBe('');
  });
});
