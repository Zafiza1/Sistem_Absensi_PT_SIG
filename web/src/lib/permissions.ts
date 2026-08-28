// Mirrors the backend's permission matrix exactly (see backend/README.md's
// "Master data" and "User management" sections) so the UI never offers an
// action the API would reject anyway. The backend is still the real
// enforcement point — this only avoids showing a button that would 403.
import type { Role } from "./types";

const WRITE_ROLES = {
  departments: ["SUPER_ADMIN", "ADMIN"],
  positions: ["SUPER_ADMIN", "ADMIN"],
  shifts: ["SUPER_ADMIN", "ADMIN", "HR"],
  employees: ["SUPER_ADMIN", "ADMIN", "HR"],
  schedules: ["SUPER_ADMIN", "ADMIN", "HR"],
  devices: ["SUPER_ADMIN", "ADMIN"],
  users: ["SUPER_ADMIN"],
} as const satisfies Record<string, readonly Role[]>;

export type WritableResource = keyof typeof WRITE_ROLES;

export function canWrite(resource: WritableResource, role: Role | undefined): boolean {
  if (!role) return false;
  return (WRITE_ROLES[resource] as readonly Role[]).includes(role);
}
