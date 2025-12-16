/** Money is always integer cents. Dates are YYYY-MM-DD strings. */
export type Cents = number;
export type ISODate = string;

export interface User {
  id: string;
  email: string;
  name: string;
  created_at: string;
}

export interface AuthResponse {
  token: string;
  expires_at: string;
  user: User;
}

export type PropertyKind = 'single_family' | 'multi_family' | 'condo' | 'townhouse' | 'commercial' | 'other';

export interface Property {
  id: string;
  name: string;
  kind: PropertyKind;
  address_line1: string;
  address_line2: string;
  city: string;
  region: string;
  postal_code: string;
  notes: string;
  unit_count: number;
  occupied_units: number;
  created_at: string;
}

export type PropertyInput = Pick<
  Property,
  'name' | 'kind' | 'address_line1' | 'address_line2' | 'city' | 'region' | 'postal_code' | 'notes'
>;

export interface Unit {
  id: string;
  property_id: string;
  property_name: string;
  label: string;
  bedrooms: number;
  bathrooms: number;
  square_feet: number | null;
  market_rent_cents: Cents;
  current_lease_id: string | null;
  current_tenant: string | null;
}

export type UnitInput = Pick<Unit, 'label' | 'bedrooms' | 'bathrooms' | 'square_feet' | 'market_rent_cents'>;

export interface Tenant {
  id: string;
  first_name: string;
  last_name: string;
  email: string;
  phone: string;
  notes: string;
  lease_count: number;
  created_at: string;
}

export type TenantInput = Pick<Tenant, 'first_name' | 'last_name' | 'email' | 'phone' | 'notes'>;

export type LeaseStatus = 'upcoming' | 'active' | 'ended' | 'terminated';

export interface Lease {
  id: string;
  unit_id: string;
  unit_label: string;
  property_id: string;
  property_name: string;
  tenant_id: string;
  tenant_name: string;
  start_date: ISODate;
  end_date: ISODate;
  rent_cents: Cents;
  deposit_cents: Cents;
  due_day: number;
  grace_days: number;
  late_fee_cents: Cents;
  terminated_on: ISODate | null;
  notes: string;
  status: LeaseStatus;
}

export type LeaseInput = Pick<
  Lease,
  | 'unit_id'
  | 'tenant_id'
  | 'start_date'
  | 'end_date'
  | 'rent_cents'
  | 'deposit_cents'
  | 'due_day'
  | 'grace_days'
  | 'late_fee_cents'
  | 'notes'
>;

export type ChargeKind = 'rent' | 'late_fee' | 'deposit' | 'utility' | 'other';
export type ChargeStatus = 'paid' | 'partial' | 'due' | 'overdue' | 'upcoming';

export interface LedgerCharge {
  id: string;
  lease_id: string;
  kind: ChargeKind;
  description: string;
  amount_cents: Cents;
  due_date: ISODate;
  period_start: ISODate | null;
  period_end: ISODate | null;
  paid_cents: Cents;
  remaining_cents: Cents;
  status: ChargeStatus;
}

export type PaymentMethod = 'bank_transfer' | 'check' | 'cash' | 'card' | 'other';

export interface Payment {
  id: string;
  lease_id: string;
  tenant_name: string;
  property_id: string;
  property_name: string;
  unit_label: string;
  amount_cents: Cents;
  paid_on: ISODate;
  method: PaymentMethod;
  reference: string;
  notes: string;
}

export interface PaymentInput {
  amount_cents: Cents;
  paid_on: ISODate;
  method: PaymentMethod;
  reference: string;
  notes: string;
}

export interface ChargeInput {
  kind: Exclude<ChargeKind, 'rent'>;
  description: string;
  amount_cents: Cents;
  due_date: ISODate;
}

export interface Ledger {
  as_of: ISODate;
  charged_cents: Cents;
  paid_cents: Cents;
  balance_cents: Cents;
  overdue_cents: Cents;
  oldest_overdue: ISODate | null;
  days_overdue: number;
  charges: LedgerCharge[];
  payments: Payment[];
}

export interface Balance {
  lease: Lease;
  balance_cents: Cents;
  overdue_cents: Cents;
  oldest_overdue: ISODate | null;
  days_overdue: number;
}

export interface PaymentList {
  payments: Payment[];
  total: number;
  page_total_cents: Cents;
}
