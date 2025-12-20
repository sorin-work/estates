import type { ChargeKind, ChargeStatus, LeaseStatus, PaymentMethod, PropertyKind } from '../api/types';

export const propertyKinds: Record<PropertyKind, string> = {
  single_family: 'Single family',
  multi_family: 'Multi-family',
  condo: 'Condo',
  townhouse: 'Townhouse',
  commercial: 'Commercial',
  other: 'Other',
};

export const leaseStatuses: Record<LeaseStatus, string> = {
  upcoming: 'Upcoming',
  active: 'Active',
  ended: 'Ended',
  terminated: 'Terminated',
};

export const chargeKinds: Record<ChargeKind, string> = {
  rent: 'Rent',
  late_fee: 'Late fee',
  deposit: 'Deposit',
  utility: 'Utility',
  other: 'Other',
};

export const chargeStatuses: Record<ChargeStatus, string> = {
  paid: 'Paid',
  partial: 'Part paid',
  due: 'Due',
  overdue: 'Overdue',
  upcoming: 'Upcoming',
};

export const paymentMethods: Record<PaymentMethod, string> = {
  bank_transfer: 'Bank transfer',
  check: 'Check',
  cash: 'Cash',
  card: 'Card',
  other: 'Other',
};

export function options<T extends string>(labels: Record<T, string>): { value: T; label: string }[] {
  return (Object.keys(labels) as T[]).map((value) => ({ value, label: labels[value] }));
}
