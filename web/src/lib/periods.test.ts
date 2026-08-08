import { describe, expect, it } from 'vitest';
import { presetRange } from './periods';

describe('presetRange', () => {
  it('works out the dates of each preset', () => {
    expect(presetRange('this-year', '2026-09-14')).toEqual({ from: '2026-01-01', to: '2026-09-14' });
    expect(presetRange('last-year', '2026-09-14')).toEqual({ from: '2025-01-01', to: '2025-12-31' });
    expect(presetRange('last-12-months', '2026-09-14')).toEqual({ from: '2025-10-01', to: '2026-09-30' });
  });
});
