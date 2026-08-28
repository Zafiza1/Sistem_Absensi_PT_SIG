import {
  LayoutDashboard,
  Users,
  Building2,
  Briefcase,
  Clock,
  CalendarDays,
  Tablet,
  ClipboardList,
  BarChart3,
  UserCog,
  ScrollText,
  type LucideIcon,
} from "lucide-react";

import type { Role } from "@/lib/types";

export interface NavItem {
  href: string;
  label: string;
  icon: LucideIcon;
  // Undefined means every authenticated role can see it (GET on these
  // resources is open to any role — see backend/README.md). Only
  // /users and /audit-logs are restricted at the API level.
  roles?: Role[];
}

export const NAV_ITEMS: NavItem[] = [
  { href: "/dashboard", label: "Dashboard", icon: LayoutDashboard },
  { href: "/employees", label: "Karyawan", icon: Users },
  { href: "/departments", label: "Departemen", icon: Building2 },
  { href: "/positions", label: "Jabatan", icon: Briefcase },
  { href: "/shifts", label: "Shift", icon: Clock },
  { href: "/schedules", label: "Jadwal Kerja", icon: CalendarDays },
  { href: "/devices", label: "Perangkat", icon: Tablet },
  { href: "/attendance", label: "Riwayat Absensi", icon: ClipboardList },
  { href: "/reports", label: "Laporan", icon: BarChart3 },
  { href: "/users", label: "Kelola Akun", icon: UserCog, roles: ["SUPER_ADMIN"] },
  { href: "/audit-logs", label: "Audit Log", icon: ScrollText, roles: ["SUPER_ADMIN"] },
];
