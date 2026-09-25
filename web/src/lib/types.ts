// Shared types matching the Go backend's response shapes exactly (see
// backend/README.md's API section) — one source of truth per resource,
// imported by every page instead of each redefining its own shape.

export type Role = "SUPER_ADMIN" | "ADMIN" | "HR" | "MANAGEMENT";

export const ROLES: Role[] = ["SUPER_ADMIN", "ADMIN", "HR", "MANAGEMENT"];

export const ROLE_LABELS: Record<Role, string> = {
  SUPER_ADMIN: "Super Admin",
  ADMIN: "Admin",
  HR: "HR",
  MANAGEMENT: "Management",
};

export interface CurrentUser {
  id: string;
  name: string;
  email: string;
  role: Role;
}

export interface PaginationMeta {
  page: number;
  page_size: number;
  total_items: number;
  total_pages: number;
}

export interface ListResponse<T> {
  items: T[];
  meta: PaginationMeta;
}

export interface Department {
  id: string;
  name: string;
  description: string;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface Position {
  id: string;
  name: string;
  description: string;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface Shift {
  id: string;
  name: string;
  start_time: string; // "HH:MM"
  end_time: string; // "HH:MM"
  late_tolerance_minutes: number;
  working_duration_minutes: number;
  is_overnight: boolean;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export type EmployeeStatus = "ACTIVE" | "INACTIVE";

export interface Employee {
  id: string;
  employee_number: string;
  name: string;
  email: string;
  phone: string;
  department_id: string | null;
  department_name: string | null;
  position_id: string | null;
  position_name: string | null;
  shift_id: string | null;
  shift_name: string | null;
  device_user_id: string | null;
  biometric_id: string | null;
  status: EmployeeStatus;
  created_at: string;
  updated_at: string;
}

// One weekday in the company-wide default weekly schedule
// (GET/PUT /company-schedule). shift_id null / is_day_off true means that
// weekday is a non-working day. A weekday the backend omits entirely is
// "not configured" — attendance falls back to the employee's own shift.
export interface CompanyScheduleDay {
  day_of_week: number; // 1=Monday..7=Sunday
  shift_id: string | null;
  shift_name: string;
  is_day_off: boolean;
}

export interface WorkSchedule {
  id: string;
  employee_id: string;
  employee_name: string;
  day_of_week: number; // 1=Monday..7=Sunday
  shift_id: string;
  shift_name: string;
  created_at: string;
  updated_at: string;
}

// GET /reports/monthly — the aggregated monthly attendance report.
export type ReportDayStatus = "ON_TIME" | "LATE" | "ABSENT" | "OFF" | "PENDING";

export interface ReportDayCell {
  day: number; // day of month, 1-based
  status: ReportDayStatus;
  late_minutes: number;
  check_in_at: string | null;
  check_out_at: string | null;
}

export interface MonthlyReportEmployee {
  employee_id: string;
  employee_number: string;
  name: string;
  department_name: string;
  working_days: number; // elapsed working days = on_time + late_count + absent
  on_time: number;
  late_count: number;
  late_minutes: number; // total for the month
  absent: number;
  days: ReportDayCell[];
}

export interface MonthlyReport {
  year: number;
  month: number; // 1-12
  days_in_month: number;
  generated_at: string;
  employees: MonthlyReportEmployee[];
}

export type DeviceStatus = "ACTIVE" | "INACTIVE";
export type DeviceType = "FINGERSPOT" | "TABLET" | "OTHER";
export type ConnectionStatus = "CONNECTED" | "DISCONNECTED" | "ERROR" | "SYNCING";
export type SyncStatus = "IDLE" | "SYNCING" | "SUCCESS" | "FAILED";

export interface Device {
  id: string;
  device_name: string;
  device_code: string;
  location: string;
  status: DeviceStatus;
  device_type: DeviceType;
  serial_number: string | null;
  ip_address: string | null;
  port: number | null;
  connection_status: ConnectionStatus;
  sync_status: SyncStatus;
  error_message: string | null;
  app_version: string | null;
  device_config: Record<string, unknown> | null;
  is_online: boolean;
  is_connected: boolean;
  is_syncing: boolean;
  last_seen_at: string | null;
  last_sync_at: string | null;
  created_at: string;
  updated_at: string;
}

export type AttendanceStatus = "ON_TIME" | "LATE" | "CHECKED_OUT" | "ABSENT" | "INCOMPLETE";

export interface Attendance {
  id: string;
  employee_id: string;
  employee_name: string;
  employee_number: string;
  shift_id: string | null;
  shift_name: string | null;
  attendance_date: string;
  check_in_at: string | null;
  check_in_device_name: string | null;
  check_out_at: string | null;
  check_out_device_name: string | null;
  status: AttendanceStatus;
  late_minutes: number;
  working_duration_minutes?: number;
  created_at: string;
  updated_at: string;
}

export interface DashboardUser {
  id: string;
  name: string;
  email: string;
  role: Role;
  is_active: boolean;
  created_at: string;
  updated_at: string;
  generated_password?: string;
}

export type AuditAction = "CREATE" | "UPDATE" | "DELETE" | "LOGIN" | "LOGIN_FAILED";

export interface AuditLogEntry {
  id: string;
  actor_id: string | null;
  actor_name: string;
  actor_role: Role | "";
  action: AuditAction;
  entity_type: string;
  entity_id: string;
  description: string;
  ip_address: string;
  created_at: string;
}

// Payroll Period Types
export type PayrollPeriodStatus = "DRAFT" | "PROCESSING" | "COMPLETED" | "LOCKED";
export type PaymentStatus = "PENDING" | "PAID" | "FAILED";
export type DeductionType = "FIXED" | "PERCENTAGE" | "HALF_DAY_SALARY";
export type DeductionRuleType = "LATE" | "ABSENT" | "OTHER";

export interface PayrollPeriod {
  id: string;
  period_start: string;
  period_end: string;
  year: number;
  month: number;
  status: PayrollPeriodStatus;
  processed_at: string | null;
  processed_by: string | null;
  total_employees: number;
  total_gross_pay: number;
  total_net_pay: number;
  total_deductions: number;
  notes: string | null;
  created_at: string;
  updated_at: string;
}

export interface PayrollItem {
  id: string;
  payroll_period_id: string;
  employee_id: string;
  employee_number: string;
  employee_name: string;
  department_id: string | null;
  department_name: string;
  position_id: string | null;
  position_name: string;
  working_days: number;
  present_days: number;
  absent_days: number;
  late_days: number;
  late_minutes: number;
  leave_days: number;
  base_salary: number;
  overtime_hours: number;
  overtime_pay: number;
  allowance: number;
  bonus: number;
  other_earnings: number;
  total_earnings: number;
  late_deduction: number;
  absent_deduction: number;
  tax_deduction: number;
  insurance_deduction: number;
  other_deductions: number;
  total_deductions: number;
  gross_pay: number;
  net_pay: number;
  payment_status: PaymentStatus;
  payment_date: string | null;
  payment_method: string | null;
  payment_reference: string | null;
  notes: string | null;
  created_at: string;
  updated_at: string;
}

export interface DeductionRule {
  id: string;
  rule_type: DeductionRuleType;
  rule_name: string;
  description: string | null;
  late_min_minutes: number | null;
  late_max_minutes: number | null;
  deduction_amount: number;
  deduction_type: DeductionType;
  percentage_value: number | null;
  is_active: boolean;
  effective_date: string;
  expiry_date: string | null;
  priority: number;
  created_at: string;
  updated_at: string;
}
