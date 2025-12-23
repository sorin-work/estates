import { chargeStatuses, leaseStatuses } from '../lib/labels';
import type { ChargeStatus, LeaseStatus } from '../api/types';

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
export function LeaseStatusBadge({ status }: { status: LeaseStatus }) {
  return <Badge tone={leaseTones[status]}>{leaseStatuses[status]}</Badge>;
}

export function ChargeStatusBadge({ status }: { status: ChargeStatus }) {
  return <Badge tone={chargeTones[status]}>{chargeStatuses[status]}</Badge>;
}
