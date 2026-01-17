import type {
  ChargeKind,
  ChargeStatus,
  ExpenseCategory,
  LeaseStatus,
  MaintenancePriority,
  MaintenanceStatus,
  PaymentMethod,
  PropertyKind,
} from '../api/types';

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

export const maintenanceStatuses: Record<MaintenanceStatus, string> = {
  open: 'Open',
  in_progress: 'In progress',
  on_hold: 'On hold',
  completed: 'Completed',
  cancelled: 'Cancelled',
};

export const maintenancePriorities: Record<MaintenancePriority, string> = {
  low: 'Low',
  normal: 'Normal',
  high: 'High',
  urgent: 'Urgent',
};

export const expenseCategories: Record<ExpenseCategory, string> = {
  repairs: 'Repairs',
  maintenance: 'Maintenance',
  utilities: 'Utilities',
  insurance: 'Insurance',
  taxes: 'Property tax',
  mortgage: 'Mortgage',
  management: 'Management',
  hoa: 'HOA',
  legal: 'Legal',
  other: 'Other',
};

/** Verb for the button that moves a request to a status. */
export const maintenanceActions: Record<MaintenanceStatus, string> = {
  open: 'Reopen',
  in_progress: 'Start work',
  on_hold: 'Put on hold',
  completed: 'Mark complete',
  cancelled: 'Cancel',
};

export function options<T extends string>(labels: Record<T, string>): { value: T; label: string }[] {
  return (Object.keys(labels) as T[]).map((value) => ({ value, label: labels[value] }));
}
