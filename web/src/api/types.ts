/** Money is always integer cents. Dates are YYYY-MM-DD strings. */
export type Cents = number;
export type ISODate = string;

export interface User {
  id: string;
  email: string;
  name: string;
  currency: string;
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
  kind: Exclude<ChargeKind, 'rent' | 'deposit'>;
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

export type MaintenanceStatus = 'open' | 'in_progress' | 'on_hold' | 'completed' | 'cancelled';
export type MaintenancePriority = 'low' | 'normal' | 'high' | 'urgent';

export interface MaintenanceRequest {
  id: string;
  property_id: string;
  property_name: string;
  unit_id: string | null;
  unit_label: string | null;
  title: string;
  description: string;
  priority: MaintenancePriority;
  status: MaintenanceStatus;
  vendor: string;
  reported_on: ISODate;
  completed_on: ISODate | null;
  cost_cents: Cents;
  next_statuses: MaintenanceStatus[];
}

export interface MaintenanceInput {
  property_id: string;
  unit_id: string | null;
  title: string;
  description: string;
  priority: MaintenancePriority;
  vendor: string;
  reported_on: ISODate;
}

export type ExpenseCategory =
  | 'repairs'
  | 'maintenance'
  | 'utilities'
  | 'insurance'
  | 'taxes'
  | 'mortgage'
  | 'management'
  | 'hoa'
  | 'legal'
  | 'other';

export interface Expense {
  id: string;
  property_id: string;
  property_name: string;
  maintenance_request_id: string | null;
  maintenance_title: string | null;
  category: ExpenseCategory;
  amount_cents: Cents;
  spent_on: ISODate;
  vendor: string;
  description: string;
}

export interface ExpenseInput {
  property_id: string;
  maintenance_request_id: string | null;
  category: ExpenseCategory;
  amount_cents: Cents;
  spent_on: ISODate;
  vendor: string;
  description: string;
}

export interface ExpenseList {
  expenses: Expense[];
  total_cents: Cents;
  by_category: { category: ExpenseCategory; amount_cents: Cents }[];
}

export interface MonthTotal {
  month: string;
  charged_cents: Cents;
  collected_cents: Cents;
  expenses_cents: Cents;
}

export interface Dashboard {
  as_of: ISODate;
  occupancy: { units: number; occupied: number; vacant: number; rate: number };
  this_month: MonthTotal;
  outstanding: { balance_cents: Cents; overdue_cents: Cents; overdue_leases: number };
  deposits: { held_cents: Cents; leases: number };
  overdue: Balance[];
  expiring_leases: Lease[];
  maintenance: {
    open: number;
    by_priority: Record<MaintenancePriority, number>;
    recent: MaintenanceRequest[];
  };
  monthly: MonthTotal[];
}

export type DepositKind = 'received' | 'returned' | 'withheld';

export interface DepositTransaction {
  id: string;
  lease_id: string;
  kind: DepositKind;
  amount_cents: Cents;
  occurred_on: ISODate;
  notes: string;
  created_at: string;
}

export interface DepositInput {
  kind: DepositKind;
  amount_cents: Cents;
  occurred_on: ISODate;
  notes: string;
}

export interface Deposit {
  lease_id: string;
  agreed_cents: Cents;
  received_cents: Cents;
  returned_cents: Cents;
  withheld_cents: Cents;
  held_cents: Cents;
  transactions: DepositTransaction[];
}

export type VendorTrade =
  | 'plumbing'
  | 'electrical'
  | 'hvac'
  | 'appliances'
  | 'carpentry'
  | 'painting'
  | 'cleaning'
  | 'landscaping'
  | 'roofing'
  | 'pest_control'
  | 'locksmith'
  | 'handyman'
  | 'other';

export interface Vendor {
  id: string;
  name: string;
  trade: VendorTrade;
  contact_name: string;
  phone: string;
  email: string;
  notes: string;
  created_at: string;
  open_jobs: number;
  spent_cents: Cents;
  last_used_on: ISODate | null;
}

export interface VendorInput {
  name: string;
  trade: VendorTrade;
  contact_name: string;
  phone: string;
  email: string;
  notes: string;
}
