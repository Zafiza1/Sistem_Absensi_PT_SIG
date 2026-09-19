"use client";

import { useEffect, useState } from "react";
import { Users, Building2, Tablet, ClipboardCheck, ArrowRight } from "lucide-react";
import Link from "next/link";

import { api } from "@/lib/api-client";
import { useAuth } from "@/lib/auth-context";
import { ROLE_LABELS } from "@/lib/types";
import type { Attendance, Department, Device, Employee, ListResponse } from "@/lib/types";

import { Card, CardContent, CardFooter } from "@/components/ui/card";
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
  href,
  tone,
}: {
  title: string;
  value: string;
  icon: React.ComponentType<{ className?: string }>;
  loading: boolean;
  hint?: string;
  href: string;
  tone: "blue" | "green" | "amber" | "violet";
}) {
  const toneClasses: Record<typeof tone, string> = {
    blue: "bg-primary/10 text-primary dark:bg-primary/20 dark:text-sky-400",
    green: "bg-emerald-50 text-emerald-600 dark:bg-emerald-500/10 dark:text-emerald-400",
    amber: "bg-amber-50 text-amber-600 dark:bg-amber-500/10 dark:text-amber-400",
    violet: "bg-violet-50 text-violet-600 dark:bg-violet-500/10 dark:text-violet-400",
  };

  return (
    <Link href={href} className="group">
      <Card className="transition-colors hover:border-primary/40">
        <CardContent className="flex items-start justify-between">
          <div>
            <p className="text-sm text-muted-foreground">{title}</p>
            {loading ? (
              <Skeleton className="mt-2 h-8 w-14" />
            ) : (
              <p className="mt-1 text-3xl font-semibold tracking-tight">{value}</p>
            )}
            {hint && !loading && <p className="mt-1 text-xs text-muted-foreground">{hint}</p>}
          </div>
          <div className={`flex size-10 shrink-0 items-center justify-center rounded-lg ${toneClasses[tone]}`}>
            <Icon className="size-5" />
          </div>
        </CardContent>
        <CardFooter className="gap-1 bg-transparent py-2.5 text-xs font-medium text-muted-foreground transition-colors group-hover:text-primary">
          Lihat detail
          <ArrowRight className="size-3 transition-transform group-hover:translate-x-0.5" />
        </CardFooter>
      </Card>
    </Link>
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
      {/* Navy hero banner — echoes the login page's brand panel and the
          Flutter kiosk's dark theme, so the landing page reads as this
          product's own chrome instead of a generic light admin template. */}
      <div className="relative mb-6 overflow-hidden rounded-xl bg-sidebar px-6 py-7 text-white">
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
            <p className="mt-1 text-sm text-slate-300">Ringkasan sistem absensi hari ini</p>
          </div>
          {user && (
            <div className="flex items-center gap-2 rounded-lg bg-white/10 px-3 py-2 text-sm">
              <span className="text-slate-300">{user.email}</span>
              <Badge className="border-sky-400/30 bg-sky-400/15 text-sky-300" variant="outline">
                {ROLE_LABELS[user.role]}
              </Badge>
            </div>
          )}
        </div>
      </div>

      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <StatCard
          title="Karyawan Aktif"
          value={String(stats?.activeEmployees ?? "-")}
          icon={Users}
          loading={loading}
          href="/employees"
          tone="blue"
        />
        <StatCard
          title="Divisi"
          value={String(stats?.departments ?? "-")}
          icon={Building2}
          loading={loading}
          href="/departments"
          tone="violet"
        />
        <StatCard
          title="Perangkat Online"
          value={stats ? `${stats.onlineDevices}/${stats.totalDevices}` : "-"}
          icon={Tablet}
          loading={loading}
          href="/devices"
          tone="amber"
        />
        <StatCard
          title="Absen Hari Ini"
          value={String(stats?.todayCheckedIn ?? "-")}
          icon={ClipboardCheck}
          loading={loading}
          hint={stats ? `${stats.todayOnTime} tepat waktu, ${stats.todayLate} terlambat` : undefined}
          href="/attendance"
          tone="green"
        />
      </div>
    </div>
  );
}
