import { afterEach, describe, expect, it, vi } from 'vitest';
import { accountZoneHint, isoDateIn, setAccountTimeZone, today } from './dates';

afterEach(() => {
  setAccountTimeZone('');
  vi.useRealTimers();
  vi.restoreAllMocks();
});

function browserIn(zone: string) {
  vi.spyOn(Intl.DateTimeFormat.prototype, 'resolvedOptions').mockReturnValue({
    timeZone: zone,
  } as Intl.ResolvedDateTimeFormatOptions);
}

describe('the account time zone', () => {
  it('works out a date in a zone', () => {
    const at = new Date('2026-03-31T23:30:00Z');
    expect(isoDateIn(at, 'Pacific/Auckland')).toBe('2026-04-01');
    expect(isoDateIn(at, 'America/Los_Angeles')).toBe('2026-03-31');
  });

  it("uses the account's zone for today", () => {
    vi.useFakeTimers({ now: new Date('2026-03-31T23:30:00Z'), toFake: ['Date'] });
    setAccountTimeZone('Pacific/Auckland');
    expect(today()).toBe('2026-04-01');
    setAccountTimeZone('America/Los_Angeles');
    expect(today()).toBe('2026-03-31');
  });

  it('falls back to the browser for an unknown zone', () => {
    setAccountTimeZone('Mars/Olympus');
    expect(today()).toMatch(/^\d{4}-\d{2}-\d{2}$/);
  });

  it('hints only when the device is in another zone', () => {
    setAccountTimeZone('Europe/Bucharest');
    browserIn('Europe/Bucharest');
    expect(accountZoneHint()).toBeUndefined();
    vi.restoreAllMocks();
    browserIn('America/New_York');
    expect(accountZoneHint()).toMatch(/Europe \/ Bucharest, your account's time zone/);
  });
});
