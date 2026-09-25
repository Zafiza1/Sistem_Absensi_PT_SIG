import {
  LayoutDashboard,
  Users,
  Building2,
  Briefcase,
  Clock,
  CalendarClock,
  CalendarDays,
  Fingerprint,
  ClipboardList,
  BarChart3,
  UserCog,
  ScrollText,
  DollarSign,
  type LucideIcon,
} from "lucide-react";

import type { Role } from "@/lib/types";

// Shared with PageHeader, which looks up the current route's icon/tone
// here so every page gets a branded header for free instead of each page
// hand-picking (and inevitably drifting on) its own icon and color.
export type NavTone = "blue" | "violet" | "amber" | "cyan" | "green" | "rose";

// Sidebar section grouping — turns one flat 12-item list into a nav with
// hierarchy, which reads as a purpose-built product instead of a generic
// template's undifferentiated link dump.
export type NavSection = "Utama" | "Data Master" | "Operasional" | "Laporan & Administrasi";

export interface NavItem {
  href: string;
  label: string;
  icon: LucideIcon;
  tone: NavTone;
  section: NavSection;
  // Undefined means every authenticated role can see it (GET on these
  // resources is open to any role — see backend/README.md). Only
  // /users and /audit-logs are restricted at the API level.
  roles?: Role[];
}

export const NAV_ITEMS: NavItem[] = [
  { href: "/dashboard", label: "Dashboard", icon: LayoutDashboard, tone: "blue", section: "Utama" },
  { href: "/employees", label: "Karyawan", icon: Users, tone: "blue", section: "Data Master" },
  { href: "/departments", label: "Divisi", icon: Building2, tone: "violet", section: "Data Master" },
  { href: "/positions", label: "Jabatan", icon: Briefcase, tone: "violet", section: "Data Master" },
  { href: "/shifts", label: "Shift", icon: Clock, tone: "amber", section: "Data Master" },
  { href: "/company-schedule", label: "Jam Kerja", icon: CalendarClock, tone: "amber", section: "Operasional" },
  { href: "/schedules", label: "Jadwal Kerja", icon: CalendarDays, tone: "amber", section: "Operasional" },
  { href: "/devices", label: "Perangkat", icon: Fingerprint, tone: "cyan", section: "Operasional" },
  { href: "/attendance", label: "Riwayat Absensi", icon: ClipboardList, tone: "green", section: "Operasional" },
  { href: "/reports", label: "Laporan", icon: BarChart3, tone: "blue", section: "Laporan & Administrasi" },
  {
    href: "/payroll",
    label: "Payroll",
    icon: DollarSign,
    tone: "green",
    section: "Laporan & Administrasi",
    roles: ["SUPER_ADMIN", "ADMIN", "HR"],
  },
  {
    href: "/users",
    label: "Kelola Akun",
    icon: UserCog,
    tone: "rose",
    section: "Laporan & Administrasi",
    roles: ["SUPER_ADMIN"],
  },
  {
    href: "/audit-logs",
    label: "Audit Log",
    icon: ScrollText,
    tone: "rose",
    section: "Laporan & Administrasi",
    roles: ["SUPER_ADMIN"],
  },
];
