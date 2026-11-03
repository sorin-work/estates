import { describe, expect, it } from 'vitest';
import { nextSort, sortBy } from './sort';

describe('sortBy', () => {
  const rows = [
    { name: 'Élena', unit: '203', owed: 0 },
    { name: 'aisha', unit: '103', owed: 162500 },
    { name: 'Tom', unit: '102', owed: 279000 },
    { name: 'daniel', unit: '201', owed: 0 },
  ];

  it('sorts strings ignoring case and accents, numbers by value', () => {
    expect(sortBy(rows, (r) => r.name, 'asc').map((r) => r.name)).toEqual(['aisha', 'daniel', 'Élena', 'Tom']);
    expect(sortBy(rows, (r) => r.owed, 'desc').map((r) => r.name)).toEqual(['Tom', 'aisha', 'Élena', 'daniel']);
  });

  it('keeps ties in their original order', () => {
    expect(sortBy(rows, (r) => r.owed, 'asc').map((r) => r.name)).toEqual(['Élena', 'daniel', 'aisha', 'Tom']);
  });
});

describe('nextSort', () => {
  it('flips the direction on a second click', () => {
    let s = nextSort({ key: null, dir: 'asc' }, 'owed', 'desc');
    expect(s).toEqual({ key: 'owed', dir: 'desc' });
    s = nextSort(s, 'owed');
    expect(s).toEqual({ key: 'owed', dir: 'asc' });
    expect(nextSort(s, 'name')).toEqual({ key: 'name', dir: 'asc' });
  });
});
