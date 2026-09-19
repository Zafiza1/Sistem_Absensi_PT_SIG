"use client";

import { useEffect, useState } from "react";
import {
  Users,
  Building2,
  Tablet,
  ArrowRight,
  ChevronRight,
  Clock,
  WifiOff,
  CheckCircle2,
} from "lucide-react";
import Link from "next/link";

import { api } from "@/lib/api-client";
import { useAuth } from "@/lib/auth-context";
import { ROLE_LABELS } from "@/lib/types";
import type { Attendance, AttendanceStatus, Department, Device, Employee, ListResponse } from "@/lib/types";
import { cn } from "@/lib/utils";

import { Card } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { Badge } from "@/components/ui/badge";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";

interface DashboardData {
  activeEmployees: number;
  departments: number;
  devices: Device[];
  attendanceToday: Attendance[];
}

const STATUS_LABELS: Record<AttendanceStatus, string> = {
  ON_TIME: "Tepat Waktu",
  LATE: "Terlambat",
  CHECKED_OUT: "Selesai",
  ABSENT: "Tidak Hadir",
  INCOMPLETE: "Belum Lengkap",
};

function todayIso(): string {
  return new Date().toLocaleDateString("en-CA"); // YYYY-MM-DD in local time
}

function initials(name: string): string {
  const parts = name.trim().split(/\s+/);
  if (parts.length === 0 || !parts[0]) return "?";
  if (parts.length === 1) return parts[0].slice(0, 1).toUpperCase();
  return (parts[0][0] + parts[parts.length - 1][0]).toUpperCase();
}

function formatTime(value: string | null): string {
  if (!value) return "-";
  return new Date(value).toLocaleTimeString("id-ID", { hour: "2-digit", minute: "2-digit" });
}

// A CHECKED_OUT record's original punctuality is only recoverable from
// late_minutes — the backend overwrites ON_TIME/LATE with CHECKED_OUT on
// checkout (backend/internal/attendance/repository.go), so counting just
// the ON_TIME/LATE labels undercounts anyone who has already gone home.
function isLate(a: Attendance): boolean {
  return a.status === "LATE" || (a.status === "CHECKED_OUT" && a.late_minutes > 0);
}

function MiniStat({
  label,
  value,
  icon: Icon,
  loading,
  href,
  tone,
  hint,
}: {
  label: string;
  value: string;
  icon: React.ComponentType<{ className?: string }>;
  loading: boolean;
  href: string;
  tone: "blue" | "amber" | "violet";
  hint?: string;
}) {
  const toneClasses: Record<typeof tone, string> = {
    blue: "bg-primary/10 text-primary dark:bg-primary/20 dark:text-sky-400",
    amber: "bg-amber-50 text-amber-600 dark:bg-amber-500/10 dark:text-amber-400",
    violet: "bg-violet-50 text-violet-600 dark:bg-violet-500/10 dark:text-violet-400",
  };

  return (
    <Link
      href={href}
      className="group flex items-center gap-3 rounded-lg px-3 py-3 transition-colors hover:bg-accent"
    >
      <div className={cn("flex size-9 shrink-0 items-center justify-center rounded-lg", toneClasses[tone])}>
        <Icon className="size-4.5" />
      </div>
      <div className="min-w-0 flex-1">
        <p className="text-xs text-muted-foreground">{label}</p>
        {loading ? (
          <Skeleton className="mt-1 h-5 w-12" />
        ) : (
          <p className="text-lg leading-tight font-semibold tracking-tight">{value}</p>
        )}
        {hint && !loading && <p className="truncate text-xs text-muted-foreground">{hint}</p>}
      </div>
      <ChevronRight className="size-4 shrink-0 text-muted-foreground/50 transition-transform group-hover:translate-x-0.5 group-hover:text-primary" />
    </Link>
  );
}

export default function DashboardHomePage() {
  const { user } = useAuth();
  const [data, setData] = useState<DashboardData | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;

    async function load() {
      try {
        const today = todayIso();
        const [employees, departments, devices, todayAttendance] = await Promise.all([
          api.get<ListResponse<Employee>>("/employees", { status: "ACTIVE", page: 1, page_size: 1 }),
          api.get<ListResponse<Department>>("/departments", { page: 1, page_size: 1 }),
          api.get<ListResponse<Device>>("/devices", { page: 1, page_size: 100 }),
          api.get<ListResponse<Attendance>>("/attendance", { date_from: today, date_to: today, page: 1, page_size: 100 }),
        ]);

        if (cancelled) return;
        setData({
          activeEmployees: employees.meta.total_items,
          departments: departments.meta.total_items,
          devices: devices.items,
          attendanceToday: todayAttendance.items,
        });
      } catch {
        // Dashboard degrades to empty widgets on failure; not worth a
        // full-page error for a summary view.
      } finally {
        if (!cancelled) setLoading(false);
      }
    }

    load();
    return () => {
      cancelled = true;
    };
  }, []);

  const attendanceToday = data?.attendanceToday ?? [];
  const onTimeCount = attendanceToday.filter((a) => !isLate(a) && a.status !== "ABSENT").length;
  const lateCount = attendanceToday.filter(isLate).length;
  const checkedInCount = attendanceToday.length;
  const notYetCount = Math.max((data?.activeEmployees ?? 0) - checkedInCount, 0);
  const attendanceTotal = onTimeCount + lateCount + notYetCount || 1;

  const recentActivity = [...attendanceToday]
    .filter((a) => a.check_in_at)
    .sort((a, b) => new Date(b.check_in_at ?? 0).getTime() - new Date(a.check_in_at ?? 0).getTime())
    .slice(0, 6);

  const offlineDevices = (data?.devices ?? []).filter((d) => !d.is_online);
  const onlineDeviceCount = (data?.devices ?? []).length - offlineDevices.length;

  return (
    <div className="space-y-4">
      {/* Navy hero banner — echoes the login page's brand panel and the
          Flutter kiosk's dark theme, so the landing page reads as this
          product's own chrome instead of a generic light admin template. */}
      <div className="relative overflow-hidden rounded-xl bg-sidebar px-6 py-7 text-white">
        <div
          className="pointer-events-none absolute inset-0 opacity-40"
          style={{
            backgroundImage:
              "radial-gradient(circle at 12% 25%, rgba(56,189,248,0.25), transparent 45%), radial-gradient(circle at 88% 85%, rgba(56,189,248,0.15), transparent 40%)",
          }}
          aria-hidden
        />
        <div className="relative flex flex-wrap items-center justify-between gap-4">
          <div>
            {/* Full name, not a first-name split: dashboard accounts are
                often role-style names ("Super Admin PT SIG"), not personal
                "First Last" names, so splitting on the first space would
                cut those off mid-title instead of shortening them
                meaningfully. */}
            <h1 className="text-2xl font-semibold tracking-tight text-balance">
              Selamat datang, {user?.name ?? ""}
            </h1>
            <p className="mt-1 text-sm text-slate-300">
              {new Date().toLocaleDateString("id-ID", { weekday: "long", day: "numeric", month: "long", year: "numeric" })}
            </p>
          </div>
          {user && (
            <Badge className="border-sky-400/30 bg-sky-400/15 px-3 py-1.5 text-sky-300" variant="outline">
              {ROLE_LABELS[user.role]}
            </Badge>
          )}
        </div>
      </div>

      <div className="grid gap-4 lg:grid-cols-3">
        {/* Kehadiran Hari Ini — the number that actually changes minute to
            minute, so it gets the largest, most detailed treatment rather
            than being one interchangeable box among four equal stat cards. */}
        <Card className="gap-4 p-5 lg:col-span-2">
          <div className="flex flex-wrap items-center justify-between gap-2">
            <div>
              <h2 className="text-sm font-semibold">Kehadiran Hari Ini</h2>
              <p className="text-xs text-muted-foreground">Ringkasan absensi karyawan aktif</p>
            </div>
            <Link
              href="/attendance"
              className="flex items-center gap-1 text-xs font-medium text-primary hover:underline"
            >
              Lihat semua
              <ArrowRight className="size-3" />
            </Link>
          </div>

          {loading ? (
            <Skeleton className="h-8 w-24" />
          ) : (
            <p className="text-3xl font-semibold tracking-tight">
              {checkedInCount}
              <span className="ml-1.5 text-base font-normal text-muted-foreground">
                / {data?.activeEmployees ?? 0} sudah absen
              </span>
            </p>
          )}

          {!loading && (
            <div
              className="flex h-2.5 w-full overflow-hidden rounded-full bg-muted"
              role="img"
              aria-label={`${onTimeCount} tepat waktu, ${lateCount} terlambat, ${notYetCount} belum absen`}
            >
              {onTimeCount > 0 && (
                <div className="h-full bg-emerald-500" style={{ width: `${(onTimeCount / attendanceTotal) * 100}%` }} />
              )}
              {lateCount > 0 && (
                <div className="h-full bg-amber-500" style={{ width: `${(lateCount / attendanceTotal) * 100}%` }} />
              )}
              {notYetCount > 0 && (
                <div className="h-full bg-border" style={{ width: `${(notYetCount / attendanceTotal) * 100}%` }} />
              )}
            </div>
          )}

          <div className="grid grid-cols-3 gap-3 text-sm">
            <div className="flex items-center gap-2">
              <span className="size-2 shrink-0 rounded-full bg-emerald-500" aria-hidden />
              <span className="text-muted-foreground">Tepat waktu</span>
              <span className="ml-auto font-semibold">{loading ? "-" : onTimeCount}</span>
            </div>
            <div className="flex items-center gap-2">
              <span className="size-2 shrink-0 rounded-full bg-amber-500" aria-hidden />
              <span className="text-muted-foreground">Terlambat</span>
              <span className="ml-auto font-semibold">{loading ? "-" : lateCount}</span>
            </div>
            <div className="flex items-center gap-2">
              <span className="size-2 shrink-0 rounded-full bg-border" aria-hidden />
              <span className="text-muted-foreground">Belum absen</span>
              <span className="ml-auto font-semibold">{loading ? "-" : notYetCount}</span>
            </div>
          </div>
        </Card>

        {/* Compact stat rows instead of three more equal-sized cards — these
            numbers barely move day to day, so they don't need the same
            visual weight as attendance. */}
        <Card className="justify-center gap-0.5 p-2">
          <MiniStat
            label="Karyawan Aktif"
            value={String(data?.activeEmployees ?? "-")}
            icon={Users}
            loading={loading}
            href="/employees"
            tone="blue"
          />
          <MiniStat
            label="Divisi"
            value={String(data?.departments ?? "-")}
            icon={Building2}
            loading={loading}
            href="/departments"
            tone="violet"
          />
          <MiniStat
            label="Perangkat Online"
            value={data ? `${onlineDeviceCount}/${data.devices.length}` : "-"}
            icon={Tablet}
            loading={loading}
            href="/devices"
            tone="amber"
            hint={
              offlineDevices.length > 0
                ? `Offline: ${offlineDevices.map((d) => d.device_name).slice(0, 2).join(", ")}${offlineDevices.length > 2 ? ` +${offlineDevices.length - 2}` : ""}`
                : undefined
            }
          />
        </Card>
      </div>

      {/* Aktivitas Terbaru — real, changing data instead of a static
          summary, so the dashboard reads as a live operations view rather
          than a template landing page. */}
      <Card className="gap-3 p-5">
        <div className="flex items-center justify-between">
          <h2 className="text-sm font-semibold">Aktivitas Absen Terbaru</h2>
          <Link
            href="/attendance"
            className="flex items-center gap-1 text-xs font-medium text-primary hover:underline"
          >
            Lihat semua
            <ArrowRight className="size-3" />
          </Link>
        </div>

        {loading ? (
          <div className="space-y-3">
            {Array.from({ length: 4 }).map((_, i) => (
              <div key={i} className="flex items-center gap-3">
                <Skeleton className="size-8 rounded-full" />
                <Skeleton className="h-4 flex-1" />
                <Skeleton className="h-4 w-14" />
              </div>
            ))}
          </div>
        ) : recentActivity.length === 0 ? (
          <div className="flex flex-col items-center gap-2 py-8 text-center text-sm text-muted-foreground">
            <Clock className="size-6 text-muted-foreground/50" />
            Belum ada karyawan yang absen hari ini
          </div>
        ) : (
          <div className="divide-y">
            {recentActivity.map((a) => (
              <div key={a.id} className="flex items-center gap-3 py-2.5 first:pt-0 last:pb-0">
                <Avatar className="size-8 shrink-0">
                  <AvatarFallback className="bg-primary/10 text-xs font-semibold text-primary">
                    {initials(a.employee_name)}
                  </AvatarFallback>
                </Avatar>
                <div className="min-w-0 flex-1">
                  <p className="truncate text-sm font-medium">{a.employee_name}</p>
                  <p className="truncate text-xs text-muted-foreground">
                    {a.employee_number}
                    {a.shift_name ? ` · ${a.shift_name}` : ""}
                  </p>
                </div>
                <div className="flex shrink-0 flex-col items-end gap-1">
                  <span className="flex items-center gap-1 text-xs text-muted-foreground">
                    <Clock className="size-3" />
                    {formatTime(a.check_in_at)}
                  </span>
                  <Badge variant={isLate(a) ? "destructive" : "secondary"} className="text-[11px]">
                    {isLate(a) ? STATUS_LABELS.LATE : a.status === "CHECKED_OUT" ? STATUS_LABELS.CHECKED_OUT : STATUS_LABELS.ON_TIME}
                  </Badge>
                </div>
              </div>
            ))}
          </div>
        )}
      </Card>

      {!loading && offlineDevices.length > 0 && (
        <Card className="gap-3 border-amber-200 bg-amber-50/60 p-4 dark:border-amber-900/40 dark:bg-amber-500/5">
          <div className="flex items-start gap-3">
            <div className="flex size-8 shrink-0 items-center justify-center rounded-lg bg-amber-100 text-amber-600 dark:bg-amber-500/10 dark:text-amber-400">
              <WifiOff className="size-4" />
            </div>
            <div className="min-w-0 flex-1">
              <p className="text-sm font-medium">
                {offlineDevices.length} perangkat sedang offline
              </p>
              <p className="mt-0.5 truncate text-xs text-muted-foreground">
                {offlineDevices.map((d) => d.device_name).join(", ")}
              </p>
            </div>
            <Link href="/devices" className="shrink-0 text-xs font-medium text-primary hover:underline">
              Cek perangkat
            </Link>
          </div>
        </Card>
      )}

      {!loading && data && offlineDevices.length === 0 && data.devices.length > 0 && (
        <p className="flex items-center gap-1.5 px-1 text-xs text-muted-foreground">
          <CheckCircle2 className="size-3.5 text-emerald-500" />
          Semua perangkat absensi online
        </p>
      )}
    </div>
  );
}
