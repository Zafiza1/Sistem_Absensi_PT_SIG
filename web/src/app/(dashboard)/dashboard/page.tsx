"use client";

import { useEffect, useState } from "react";
import {
  Users,
  CheckCircle2,
  Clock,
  XCircle,
  Fingerprint,
  WifiOff,
  ArrowRight,
} from "lucide-react";
import Link from "next/link";

import { api } from "@/lib/api-client";
import { useAuth } from "@/lib/auth-context";
import type { Attendance, AttendanceStatus, Device, Employee, ListResponse } from "@/lib/types";

import { Card } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { Badge } from "@/components/ui/badge";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from "@/components/ui/table";
import { StatCard } from "@/components/dashboard/stat-card";
import { AttendanceTrendChart, type TrendPoint } from "@/components/dashboard/attendance-trend-chart";
import { AttendanceDonut, type DonutSegment } from "@/components/dashboard/attendance-donut";
import { LiveClock } from "@/components/dashboard/live-clock";

interface DashboardData {
  totalEmployees: number;
  activeEmployees: number;
  devices: Device[];
  attendanceToday: Attendance[];
  weekAttendance: Attendance[];
}

const STATUS_LABELS: Record<AttendanceStatus, string> = {
  ON_TIME: "Hadir",
  LATE: "Terlambat",
  CHECKED_OUT: "Selesai",
  ABSENT: "Tidak Hadir",
  INCOMPLETE: "Belum Lengkap",
};

function todayIso(): string {
  return new Date().toLocaleDateString("en-CA"); // YYYY-MM-DD in local time
}

function isoDaysAgo(n: number): string {
  const d = new Date();
  d.setDate(d.getDate() - n);
  return d.toLocaleDateString("en-CA");
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

// Loops the shared page/page_size envelope (backend/pkg/pagination, capped
// at 100/page) until every row in the date range is collected — a week of
// attendance across the whole company can span several pages.
async function fetchAllAttendance(dateFrom: string, dateTo: string): Promise<Attendance[]> {
  const pageSize = 100;
  const first = await api.get<ListResponse<Attendance>>("/attendance", {
    date_from: dateFrom,
    date_to: dateTo,
    page: 1,
    page_size: pageSize,
  });
  const items = [...first.items];
  for (let page = 2; page <= first.meta.total_pages; page++) {
    const next = await api.get<ListResponse<Attendance>>("/attendance", {
      date_from: dateFrom,
      date_to: dateTo,
      page,
      page_size: pageSize,
    });
    items.push(...next.items);
  }
  return items;
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
        const weekStart = isoDaysAgo(6);
        const [totalRes, activeRes, devicesRes, attendanceToday, weekAttendance] = await Promise.all([
          api.get<ListResponse<Employee>>("/employees", { page: 1, page_size: 1 }),
          api.get<ListResponse<Employee>>("/employees", { status: "ACTIVE", page: 1, page_size: 1 }),
          api.get<ListResponse<Device>>("/devices", { page: 1, page_size: 100 }),
          fetchAllAttendance(today, today),
          fetchAllAttendance(weekStart, today),
        ]);

        if (cancelled) return;
        setData({
          totalEmployees: totalRes.meta.total_items,
          activeEmployees: activeRes.meta.total_items,
          devices: devicesRes.items,
          attendanceToday,
          weekAttendance,
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

  const activeEmployees = data?.activeEmployees ?? 0;
  const attendanceToday = data?.attendanceToday ?? [];
  const onTimeCount = attendanceToday.filter((a) => !isLate(a) && a.status !== "ABSENT").length;
  const lateCount = attendanceToday.filter(isLate).length;
  const checkedInCount = attendanceToday.length;
  const notPresentCount = Math.max(activeEmployees - checkedInCount, 0);

  const donutSegments: DonutSegment[] = [
    { label: "Hadir", value: onTimeCount, colorClass: "text-emerald-500", dotClass: "bg-emerald-500" },
    { label: "Terlambat", value: lateCount, colorClass: "text-amber-500", dotClass: "bg-amber-500" },
    { label: "Tidak Hadir", value: notPresentCount, colorClass: "text-red-500", dotClass: "bg-red-500" },
  ];

  // 7-day trend, grouped from the same range query — each day's "tidak
  // hadir" is approximated against today's active-employee count, since
  // the backend doesn't track a historical headcount snapshot per day.
  const trend: TrendPoint[] = Array.from({ length: 7 }).map((_, i) => {
    const iso = isoDaysAgo(6 - i);
    const dayRecords = (data?.weekAttendance ?? []).filter((a) => a.attendance_date === iso);
    const late = dayRecords.filter(isLate).length;
    const present = dayRecords.length;
    return {
      label: new Date(`${iso}T00:00:00`).toLocaleDateString("id-ID", { day: "numeric", month: "short" }),
      hadir: present - late,
      terlambat: late,
      tidakHadir: Math.max(activeEmployees - present, 0),
    };
  });

  const recentActivity = [...attendanceToday]
    .filter((a) => a.check_in_at)
    .sort((a, b) => new Date(b.check_in_at ?? 0).getTime() - new Date(a.check_in_at ?? 0).getTime())
    .slice(0, 6);

  const devices = data?.devices ?? [];
  const connectedDevices = devices.filter((d) => d.is_connected);
  const connectedDeviceCount = connectedDevices.length;

  const pct = (n: number) => (activeEmployees > 0 ? `${((n / activeEmployees) * 100).toFixed(2)}% dari total karyawan` : undefined);

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div>
          <h1 className="text-lg font-semibold tracking-tight">Dashboard</h1>
          <p className="text-sm text-muted-foreground">
            Selamat datang kembali, {user?.name ?? ""} 👋
          </p>
        </div>
        <p className="text-sm text-muted-foreground">
          {new Date().toLocaleDateString("id-ID", { weekday: "long", day: "numeric", month: "long", year: "numeric" })}
        </p>
      </div>

      <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <StatCard
          label="Total Karyawan"
          value={String(data?.totalEmployees ?? "-")}
          icon={Users}
          tone="blue"
          hint={data ? `Aktif ${activeEmployees}` : undefined}
          loading={loading}
        />
        <StatCard
          label="Hadir Hari Ini"
          value={String(onTimeCount)}
          icon={CheckCircle2}
          tone="emerald"
          hint={pct(onTimeCount)}
          loading={loading}
        />
        <StatCard
          label="Terlambat"
          value={String(lateCount)}
          icon={Clock}
          tone="amber"
          hint={pct(lateCount)}
          loading={loading}
        />
        <StatCard
          label="Tidak Hadir"
          value={String(notPresentCount)}
          icon={XCircle}
          tone="red"
          hint={pct(notPresentCount)}
          loading={loading}
        />
      </div>

      <div className="grid gap-4 xl:grid-cols-3">
        <Card className="gap-4 p-5 xl:col-span-2">
          <div className="flex flex-wrap items-center justify-between gap-2">
            <div>
              <h2 className="text-sm font-semibold">Grafik Kehadiran</h2>
              <p className="text-xs text-muted-foreground">7 hari terakhir</p>
            </div>
          </div>
          <AttendanceTrendChart data={trend} loading={loading} />
        </Card>

        <Card className="gap-4 p-5">
          <div>
            <h2 className="text-sm font-semibold">Ringkasan Kehadiran Hari Ini</h2>
            <p className="text-xs text-muted-foreground">Karyawan aktif</p>
          </div>
          <AttendanceDonut segments={donutSegments} total={activeEmployees} loading={loading} />
          <LiveClock />
        </Card>
      </div>

      <div className="grid gap-4 xl:grid-cols-3">
        <Card className="gap-3 p-5 xl:col-span-2">
          <div className="flex items-center justify-between">
            <h2 className="text-sm font-semibold">Absensi Terbaru</h2>
            <Link href="/attendance" className="flex items-center gap-1 text-xs font-medium text-primary hover:underline">
              Lihat Semua
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
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Karyawan</TableHead>
                  <TableHead>Waktu</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>Perangkat</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {recentActivity.map((a) => (
                  <TableRow key={a.id}>
                    <TableCell>
                      <div className="flex items-center gap-2.5">
                        <Avatar className="size-8 shrink-0">
                          <AvatarFallback className="bg-primary/10 text-xs font-semibold text-primary">
                            {initials(a.employee_name)}
                          </AvatarFallback>
                        </Avatar>
                        <div className="min-w-0">
                          <p className="truncate text-sm font-medium">{a.employee_name}</p>
                          <p className="truncate text-xs text-muted-foreground">
                            {a.employee_number}
                            {a.shift_name ? ` · ${a.shift_name}` : ""}
                          </p>
                        </div>
                      </div>
                    </TableCell>
                    <TableCell className="text-sm text-muted-foreground">{formatTime(a.check_in_at)}</TableCell>
                    <TableCell>
                      <Badge variant={isLate(a) ? "destructive" : "secondary"} className="text-[11px]">
                        {isLate(a) ? STATUS_LABELS.LATE : a.status === "CHECKED_OUT" ? STATUS_LABELS.CHECKED_OUT : STATUS_LABELS.ON_TIME}
                      </Badge>
                    </TableCell>
                    <TableCell className="text-sm text-muted-foreground">{a.check_in_device_name ?? "-"}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </Card>

        <Card className="gap-3 p-5">
          <div className="flex items-center justify-between">
            <h2 className="text-sm font-semibold">Status Perangkat</h2>
            <Link href="/devices" className="flex items-center gap-1 text-xs font-medium text-primary hover:underline">
              Lihat Semua
              <ArrowRight className="size-3" />
            </Link>
          </div>

          {loading ? (
            <div className="space-y-3">
              {Array.from({ length: 3 }).map((_, i) => (
                <Skeleton key={i} className="h-10 w-full" />
              ))}
            </div>
          ) : devices.length === 0 ? (
            <div className="flex flex-col items-center gap-2 py-8 text-center text-sm text-muted-foreground">
              <Fingerprint className="size-6 text-muted-foreground/50" />
              Belum ada perangkat terdaftar
            </div>
          ) : (
            <>
              <p className="flex items-center gap-1.5 text-xs text-muted-foreground">
                <CheckCircle2 className="size-3.5 text-emerald-500" />
                {connectedDeviceCount} dari {devices.length} perangkat terhubung
              </p>
              <div className="divide-y">
                {devices.slice(0, 5).map((d) => (
                  <div key={d.id} className="flex items-center gap-3 py-2.5 first:pt-0 last:pb-0">
                    <div
                      className={`flex size-8 shrink-0 items-center justify-center rounded-lg ${
                        d.is_connected
                          ? "bg-emerald-50 text-emerald-600 dark:bg-emerald-500/10 dark:text-emerald-400"
                          : "bg-red-50 text-red-600 dark:bg-red-500/10 dark:text-red-400"
                      }`}
                    >
                      {d.is_connected ? <Fingerprint className="size-4" /> : <WifiOff className="size-4" />}
                    </div>
                    <div className="min-w-0 flex-1">
                      <p className="truncate text-sm font-medium">{d.device_name}</p>
                      <p className="truncate text-xs text-muted-foreground">{d.location}</p>
                    </div>
                    <div className="flex items-center gap-2">
                      <Badge variant="outline" className="text-[10px]">
                        {d.device_type === "FINGERSPOT" ? "Fingerspot" : d.device_type}
                      </Badge>
                      <Badge variant={d.is_connected ? "secondary" : "destructive"} className="text-[11px]">
                        {d.is_connected ? "Terhubung" : "Terputus"}
                      </Badge>
                    </div>
                  </div>
                ))}
              </div>
            </>
          )}
        </Card>
      </div>
    </div>
  );
}
