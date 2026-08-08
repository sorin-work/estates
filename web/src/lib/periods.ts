import { addDays, addMonths } from './dates';

export type PeriodPreset = 'this-year' | 'last-year' | 'last-12-months' | 'custom';

export const periodPresets: Record<PeriodPreset, string> = {
  'this-year': 'This year',
  'last-year': 'Last year',
  'last-12-months': 'Last 12 months',
  custom: 'Custom',
};

/** The dates a preset covers, relative to today (YYYY-MM-DD). */
export function presetRange(preset: Exclude<PeriodPreset, 'custom'>, today: string): { from: string; to: string } {
  const year = Number(today.slice(0, 4));
  switch (preset) {
    case 'this-year':
      return { from: `${year}-01-01`, to: today };
    case 'last-year':
      return { from: `${year - 1}-01-01`, to: `${year - 1}-12-31` };
    case 'last-12-months':
      return {
        from: addMonths(`${today.slice(0, 7)}-01`, -11),
        to: addDays(addMonths(`${today.slice(0, 7)}-01`, 1), -1),
      };
  }
}
