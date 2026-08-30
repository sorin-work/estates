import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import ActivityList, { groupByDay } from './ActivityList';

const entry = (id: number, created_at: string, summary: string) => ({
  id,
  entity_type: 'payment',
  entity_id: 'x',
  lease_id: null,
  action: 'created',
  summary,
  created_at,
});

describe('ActivityList', () => {
  it('groups entries by day', () => {
    const entries = [
      entry(3, '2026-09-21T18:30:00', 'Payment of 1,450.00 from Grace Hopper on 2026-09-21'),
      entry(2, '2026-09-21T09:05:00', 'Added tenant Grace Hopper'),
      entry(1, '2026-09-19T20:00:00', 'Added property Cedar Flats'),
    ];
    expect(groupByDay(entries).map((g) => g.entries.length)).toEqual([2, 1]);

    render(<ActivityList entries={entries} />);
    expect(screen.getByRole('heading', { name: 'Monday, Sep 21, 2026' })).toBeInTheDocument();
    expect(screen.getByText('Added property Cedar Flats')).toBeInTheDocument();
    expect(screen.getByText('6:30 PM')).toBeInTheDocument();
  });
});
