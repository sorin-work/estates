import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it, vi } from 'vitest';
import type { MaintenanceRequest } from '../api/types';
import MaintenanceActions from './MaintenanceActions';

const request: MaintenanceRequest = {
  id: 'm1',
  property_id: 'p1',
  property_name: 'Harbor View',
  unit_id: null,
  unit_label: null,
  title: 'Hallway light out',
  description: '',
  priority: 'normal',
  status: 'open',
  vendor: '',
  reported_on: '2026-03-01',
  completed_on: null,
  cost_cents: 0,
  next_statuses: ['in_progress', 'on_hold', 'completed', 'cancelled'],
};

describe('MaintenanceActions', () => {
  it('offers the allowed moves, most likely first', () => {
    render(<MaintenanceActions request={request} onChange={() => {}} />);
    const labels = screen.getAllByRole('button').map((b) => b.textContent);
    expect(labels).toEqual(['Start work', 'Mark complete', 'Put on hold', 'Cancel']);
  });

  it('only offers reopening a completed request', async () => {
    const onChange = vi.fn();
    render(
      <MaintenanceActions request={{ ...request, status: 'completed', next_statuses: ['open'] }} onChange={onChange} />,
    );
    const buttons = screen.getAllByRole('button');
    expect(buttons).toHaveLength(1);
    await userEvent.click(buttons[0]!);
    expect(onChange).toHaveBeenCalledWith('open');
  });
});
