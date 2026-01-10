import { chargeStatuses, leaseStatuses, maintenancePriorities, maintenanceStatuses } from '../lib/labels';
import type { ChargeStatus, LeaseStatus, MaintenancePriority, MaintenanceStatus } from '../api/types';

type Tone = 'green' | 'blue' | 'amber' | 'red' | 'gray';

function Badge({ tone, children }: { tone: Tone; children: string }) {
  return <span className={`badge badge-${tone}`}>{children}</span>;
}

const leaseTones: Record<LeaseStatus, Tone> = { upcoming: 'blue', active: 'green', ended: 'gray', terminated: 'gray' };
const chargeTones: Record<ChargeStatus, Tone> = {
  paid: 'green',
  partial: 'amber',
  due: 'blue',
  overdue: 'red',
  upcoming: 'gray',
};
const maintenanceTones: Record<MaintenanceStatus, Tone> = {
  open: 'blue',
  in_progress: 'amber',
  on_hold: 'gray',
  completed: 'green',
  cancelled: 'gray',
};
const priorityTones: Record<MaintenancePriority, Tone> = { low: 'gray', normal: 'blue', high: 'amber', urgent: 'red' };

export function LeaseStatusBadge({ status }: { status: LeaseStatus }) {
  return <Badge tone={leaseTones[status]}>{leaseStatuses[status]}</Badge>;
}

export function ChargeStatusBadge({ status }: { status: ChargeStatus }) {
  return <Badge tone={chargeTones[status]}>{chargeStatuses[status]}</Badge>;
}

export function MaintenanceStatusBadge({ status }: { status: MaintenanceStatus }) {
  return <Badge tone={maintenanceTones[status]}>{maintenanceStatuses[status]}</Badge>;
}

export function PriorityBadge({ priority }: { priority: MaintenancePriority }) {
  return <Badge tone={priorityTones[priority]}>{maintenancePriorities[priority]}</Badge>;
}
