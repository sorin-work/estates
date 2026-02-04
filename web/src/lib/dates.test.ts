import { describe, expect, it } from 'vitest';
import {
  addDays,
  addMonths,
  daysBetween,
  formatDate,
  monthRange,
  parseISODate,
  relativeDays,
  toISODate,
} from './dates';

describe('dates', () => {
  it('parses ISO dates as local dates', () => {
    const d = parseISODate('2026-03-01');
    expect(d.getFullYear()).toBe(2026);
    expect(d.getMonth()).toBe(2);
    expect(d.getDate()).toBe(1);
    expect(toISODate(d)).toBe('2026-03-01');
  });

  it('formats dates for display', () => {
    expect(formatDate('2026-07-04')).toBe('Jul 4, 2026');
    expect(formatDate(null)).toBe('—');
  });

  it('adds days across month and year ends', () => {
    expect(addDays('2026-01-31', 1)).toBe('2026-02-01');
    expect(addDays('2026-12-31', 1)).toBe('2027-01-01');
    expect(addDays('2026-03-01', -1)).toBe('2026-02-28');
  });

  it('clamps the day when adding months', () => {
    expect(addMonths('2026-01-31', 1)).toBe('2026-02-28');
    expect(addMonths('2028-01-31', 1)).toBe('2028-02-29');
    expect(addMonths('2026-11-15', 2)).toBe('2027-01-15');
  });

  it('counts days between dates', () => {
    expect(daysBetween('2026-03-01', '2026-03-31')).toBe(30);
    expect(daysBetween('2026-03-31', '2026-03-01')).toBe(-30);
    // Across a daylight saving change.
    expect(daysBetween('2026-03-01', '2026-04-01')).toBe(31);
  });

  it('works out month ranges', () => {
    expect(monthRange('2026-02')).toEqual({ from: '2026-02-01', to: '2026-02-28' });
    expect(monthRange('2026-12')).toEqual({ from: '2026-12-01', to: '2026-12-31' });
  });

  it('describes relative days', () => {
    expect(relativeDays(0)).toBe('today');
    expect(relativeDays(1)).toBe('tomorrow');
    expect(relativeDays(12)).toBe('in 12 days');
    expect(relativeDays(-3)).toBe('3 days ago');
  });
});
