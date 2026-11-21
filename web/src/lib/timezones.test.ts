import { afterEach, describe, expect, it, vi } from 'vitest';
import { browserTimeZone, timeZoneLabel, timeZoneOptions } from './timezones';

afterEach(() => {
  vi.restoreAllMocks();
});

describe('time zones', () => {
  it('lists UTC first and includes the zones asked for', () => {
    const zones = timeZoneOptions('Mars/Olympus');
    expect(zones[0]).toBe('UTC');
    expect(zones).toContain('Mars/Olympus');
    expect(zones).toContain('Europe/Bucharest');
    expect(new Set(zones).size).toBe(zones.length);
  });

  it('falls back to a short list without Intl.supportedValuesOf', () => {
    vi.spyOn(Intl, 'supportedValuesOf').mockImplementation(() => {
      throw new Error('not supported');
    });
    const zones = timeZoneOptions('Pacific/Auckland');
    expect(zones).toContain('Pacific/Auckland');
    expect(zones).toContain('America/New_York');
  });

  it('reads the browser zone', () => {
    vi.spyOn(Intl.DateTimeFormat.prototype, 'resolvedOptions').mockReturnValue({
      timeZone: 'Pacific/Auckland',
    } as Intl.ResolvedDateTimeFormatOptions);
    expect(browserTimeZone()).toBe('Pacific/Auckland');
  });

  it('labels zones for people', () => {
    expect(timeZoneLabel('America/Los_Angeles')).toBe('America / Los Angeles');
  });
});
