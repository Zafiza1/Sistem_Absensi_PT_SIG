"use client";

import { useEffect, useState } from "react";
import { Users, Building2, Tablet, ClipboardCheck } from "lucide-react";

import { api } from "@/lib/api-client";
import { useAuth } from "@/lib/auth-context";
import type { Attendance, Department, Device, Employee, ListResponse } from "@/lib/types";

import { PageHeader } from "@/components/page-header";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { Badge } from "@/components/ui/badge";

interface Stats {
  activeEmployees: number;
  departments: number;
  onlineDevices: number;
  totalDevices: number;
  todayOnTime: number;
  todayLate: number;
  todayCheckedIn: number;
}

function todayIso(): string {
  return new Date().toLocaleDateString("en-CA"); // YYYY-MM-DD in local time
}

function StatCard({
  title,
  value,
  icon: Icon,
  loading,
  hint,
}: {
  title: string;
  value: string;
  icon: React.ComponentType<{ className?: string }>;
  loading: boolean;
  hint?: string;
}) {
  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
        <CardTitle className="text-sm font-medium text-muted-foreground">{title}</CardTitle>
        <Icon className="size-4 text-muted-foreground" />
      </CardHeader>
      <CardContent>
        {loading ? <Skeleton className="h-8 w-16" /> : <div className="text-2xl font-bold">{value}</div>}
        {hint && !loading && <p className="mt-1 text-xs text-muted-foreground">{hint}</p>}
      </CardContent>
    </Card>
  );
}

export default function DashboardHomePage() {
  const { user } = useAuth();
  const [stats, setStats] = useState<Stats | null>(null);
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
        setStats({
          activeEmployees: employees.meta.total_items,
          departments: departments.meta.total_items,
          onlineDevices: devices.items.filter((d) => d.is_online).length,
          totalDevices: devices.meta.total_items,
          todayOnTime: todayAttendance.items.filter((a) => a.status === "ON_TIME").length,
          todayLate: todayAttendance.items.filter((a) => a.status === "LATE").length,
          todayCheckedIn: todayAttendance.meta.total_items,
        });
      } catch {
        // Stat cards degrade to "-" on failure; not worth a full-page error
        // for a summary widget.
      } finally {
        if (!cancelled) setLoading(false);
      }
    }

    load();
    return () => {
      cancelled = true;
    };
  }, []);

  return (
    <div>
      <PageHeader title={`Selamat datang, ${user?.name ?? ""}`} description="Ringkasan sistem absensi hari ini" />

      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <StatCard title="Karyawan Aktif" value={String(stats?.activeEmployees ?? "-")} icon={Users} loading={loading} />
        <StatCard title="Departemen" value={String(stats?.departments ?? "-")} icon={Building2} loading={loading} />
        <StatCard
          title="Perangkat Online"
          value={stats ? `${stats.onlineDevices}/${stats.totalDevices}` : "-"}
          icon={Tablet}
          loading={loading}
        />
        <StatCard
          title="Absen Hari Ini"
          value={String(stats?.todayCheckedIn ?? "-")}
          icon={ClipboardCheck}
          loading={loading}
          hint={stats ? `${stats.todayOnTime} tepat waktu, ${stats.todayLate} terlambat` : undefined}
        />
      </div>

      <Card className="mt-6">
        <CardHeader>
          <CardTitle className="text-base">Akun Anda</CardTitle>
        </CardHeader>
        <CardContent className="flex items-center gap-3 text-sm">
          <span className="text-muted-foreground">Masuk sebagai</span>
          <span className="font-medium">{user?.name}</span>
          <Badge variant="outline">{user?.role}</Badge>
        </CardContent>
      </Card>
    </div>
  );
}
