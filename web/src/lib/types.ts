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
  status: EmployeeStatus;
  created_at: string;
  updated_at: string;
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

export type DeviceStatus = "ACTIVE" | "INACTIVE";

export interface Device {
  id: string;
  device_name: string;
  device_code: string;
  location: string;
  status: DeviceStatus;
  app_version: string | null;
  is_online: boolean;
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
