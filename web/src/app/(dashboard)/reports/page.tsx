"use client";

import { useEffect, useState } from "react";
import {
  Download,
  CheckCircle2,
  Clock,
  UserX,
  LogOut,
  Users,
  FileText,
  Thermometer,
  Palmtree,
  ClipboardCheck,
} from "lucide-react";
import type { LucideIcon } from "lucide-react";
import { toast } from "sonner";

import { cn } from "@/lib/utils";
import { api, ApiError, downloadFile } from "@/lib/api-client";
import { useAuth } from "@/lib/auth-context";
import { canWrite } from "@/lib/permissions";
import { useOptionsList } from "@/hooks/use-options-list";
import type {
  Attendance,
  Department,
  Employee,
  Headcount,
  Leave,
  LeaveType,
  ListResponse,
  MonthlyReport,
  YearlyReport,
} from "@/lib/types";

import { PageHeader } from "@/components/page-header";
import { EmptyState } from "@/components/empty-state";
import { ConfirmDeleteDialog } from "@/components/confirm-delete-dialog";
import { Card } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { Textarea } from "@/components/ui/textarea";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";

const ALL_DEPARTMENTS = "__all__";

const MONTH_NAMES = [
  "Januari",
  "Februari",
  "Maret",
  "April",
  "Mei",
  "Juni",
  "Juli",
  "Agustus",
  "September",
  "Oktober",
  "November",
  "Desember",
];

const LEAVE_LABELS: Record<LeaveType, string> = {
  IZIN: "Izin",
  SAKIT: "Sakit",
  CUTI: "Cuti",
};

type StatTone = "green" | "amber" | "rose" | "blue" | "violet" | "teal";

const STAT_TONE_CLASSES: Record<StatTone, string> = {
  blue: "bg-primary/10 text-primary dark:bg-primary/20 dark:text-sky-400",
  green: "bg-emerald-50 text-emerald-600 dark:bg-emerald-500/10 dark:text-emerald-400",
  amber: "bg-amber-50 text-amber-600 dark:bg-amber-500/10 dark:text-amber-400",
  rose: "bg-rose-50 text-rose-600 dark:bg-rose-500/10 dark:text-rose-400",
  violet: "bg-violet-50 text-violet-600 dark:bg-violet-500/10 dark:text-violet-400",
  teal: "bg-teal-50 text-teal-600 dark:bg-teal-500/10 dark:text-teal-400",
};

function StatTile({
  label,
  value,
  hint,
  icon: Icon,
  tone,
  loading,
}: {
  label: string;
  value: number | undefined;
  hint?: string;
  icon: LucideIcon;
  tone: StatTone;
  loading: boolean;
}) {
  return (
    <Card className="flex-row items-center gap-3 p-4">
      <div className={cn("flex size-10 shrink-0 items-center justify-center rounded-lg", STAT_TONE_CLASSES[tone])}>
        <Icon className="size-5" />
      </div>
      <div>
        <p className="text-xs text-muted-foreground">{label}</p>
        {loading ? (
          <Skeleton className="mt-1 h-6 w-10" />
        ) : (
          <p className="text-xl font-semibold tracking-tight">
            {value ?? 0}
            {hint && <span className="ml-1 text-xs font-normal text-muted-foreground">{hint}</span>}
          </p>
        )}
      </div>
    </Card>
  );
}

function DepartmentSelect({
  departments,
  value,
  onChange,
}: {
  departments: Department[];
  value: string;
  onChange: (v: string) => void;
}) {
  return (
    <div className="space-y-1.5">
      <Label className="text-xs text-muted-foreground">Divisi</Label>
      <Select
        items={{ [ALL_DEPARTMENTS]: "Semua divisi", ...Object.fromEntries(departments.map((d) => [d.id, d.name])) }}
        value={value}
        onValueChange={(v) => onChange(v ?? ALL_DEPARTMENTS)}
      >
        <SelectTrigger className="w-48">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value={ALL_DEPARTMENTS}>Semua divisi</SelectItem>
          {departments.map((d) => (
            <SelectItem key={d.id} value={d.id}>
              {d.name}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  );
}

function Count({ value, className }: { value: number; className: string }) {
  return value > 0 ? <span className={cn("font-medium", className)}>{value}</span> : <>0</>;
}

export default function ReportsPage() {
  const [mode, setMode] = useState("bulanan");

  return (
    <div>
      <PageHeader title="Laporan Kehadiran" description="Rekap kehadiran karyawan — per hari, per bulan, atau per tahun" />

      <Tabs value={mode} onValueChange={(v) => setMode(String(v))}>
        <TabsList className="mb-4">
          <TabsTrigger value="bulanan">Bulanan</TabsTrigger>
          <TabsTrigger value="harian">Harian</TabsTrigger>
          <TabsTrigger value="tahunan">Tahunan</TabsTrigger>
        </TabsList>
        <TabsContent value="bulanan">
          <MonthlyReportView />
        </TabsContent>
        <TabsContent value="harian">
          <DailyReportView />
        </TabsContent>
        <TabsContent value="tahunan">
          <YearlyReportView />
        </TabsContent>
      </Tabs>
    </div>
  );
}

// --- Monthly ---------------------------------------------------------------

function currentMonth(): string {
  const d = new Date();
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}`;
}

function MonthlyReportView() {
  const [month, setMonth] = useState(currentMonth());
  const [departmentId, setDepartmentId] = useState(ALL_DEPARTMENTS);
  const [report, setReport] = useState<MonthlyReport | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [exporting, setExporting] = useState(false);

  const { options: departments } = useOptionsList<Department>("/departments");
  const deptQuery = departmentId === ALL_DEPARTMENTS ? undefined : departmentId;

  useEffect(() => {
    let cancelled = false;
    // eslint-disable-next-line react-hooks/set-state-in-effect
    setLoading(true);
    setError(null);

    api
      .get<MonthlyReport>("/reports/monthly", { month, department_id: deptQuery })
      .then((res) => {
        if (!cancelled) setReport(res);
      })
      .catch((err: unknown) => {
        if (!cancelled) setError(err instanceof ApiError ? err.message : "Gagal memuat laporan bulanan");
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });

    return () => {
      cancelled = true;
    };
  }, [month, deptQuery]);

  async function handleExport() {
    setExporting(true);
    try {
      await downloadFile("/reports/monthly", { month, department_id: deptQuery, format: "xlsx" });
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "Gagal mengunduh file Excel");
    } finally {
      setExporting(false);
    }
  }

  const totals = report?.employees.reduce(
    (acc, e) => {
      acc.on_time += e.on_time;
      acc.late_count += e.late_count;
      acc.izin += e.izin;
      acc.sakit += e.sakit;
      acc.cuti += e.cuti;
      acc.absent += e.absent;
      return acc;
    },
    { on_time: 0, late_count: 0, izin: 0, sakit: 0, cuti: 0, absent: 0 },
  );

  const cards: { label: string; value: number | undefined; icon: LucideIcon; tone: StatTone }[] = [
    { label: "Tepat Waktu", value: totals?.on_time, icon: CheckCircle2, tone: "green" },
    { label: "Terlambat (kali)", value: totals?.late_count, icon: Clock, tone: "amber" },
    { label: "Izin (hari)", value: totals?.izin, icon: FileText, tone: "blue" },
    { label: "Sakit (hari)", value: totals?.sakit, icon: Thermometer, tone: "violet" },
    { label: "Cuti (hari)", value: totals?.cuti, icon: Palmtree, tone: "teal" },
    { label: "Tidak Hadir (hari)", value: totals?.absent, icon: UserX, tone: "rose" },
  ];

  return (
    <div>
      <div className="mb-4 flex flex-wrap items-end gap-3 rounded-lg border bg-card p-4">
        <div className="space-y-1.5">
          <Label className="text-xs text-muted-foreground">Bulan</Label>
          <Input type="month" value={month} onChange={(e) => setMonth(e.target.value)} className="w-44" />
        </div>
        <DepartmentSelect departments={departments} value={departmentId} onChange={setDepartmentId} />
        <Button
          variant="outline"
          onClick={handleExport}
          disabled={exporting || loading || !report || report.employees.length === 0}
          className="ml-auto"
        >
          <Download className="size-4" />
          {exporting ? "Menyiapkan..." : "Export Excel"}
        </Button>
      </div>

      <div className="mb-6 grid gap-4 sm:grid-cols-3 xl:grid-cols-6">
        {cards.map((c) => (
          <StatTile key={c.label} label={c.label} value={c.value} icon={c.icon} tone={c.tone} loading={loading} />
        ))}
      </div>

      <div className="overflow-x-auto rounded-lg border bg-card">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>NIK</TableHead>
              <TableHead>Nama</TableHead>
              <TableHead>Divisi</TableHead>
              <TableHead className="text-right">Hari Kerja</TableHead>
              <TableHead className="text-right">Tepat Waktu</TableHead>
              <TableHead className="text-right">Terlambat</TableHead>
              <TableHead className="text-right">Menit Terlambat</TableHead>
              <TableHead className="text-right">Izin</TableHead>
              <TableHead className="text-right">Sakit</TableHead>
              <TableHead className="text-right">Cuti</TableHead>
              <TableHead className="text-right">Tidak Hadir</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {loading &&
              Array.from({ length: 6 }).map((_, i) => (
                <TableRow key={i}>
                  <TableCell colSpan={11}>
                    <Skeleton className="h-5 w-full" />
                  </TableCell>
                </TableRow>
              ))}
            {!loading && error && (
              <TableRow>
                <TableCell colSpan={11} className="py-8 text-center text-destructive">
                  {error}
                </TableCell>
              </TableRow>
            )}
            {!loading && !error && report?.employees.length === 0 && (
              <TableRow>
                <TableCell colSpan={11}>
                  <EmptyState icon={Users} title="Tidak ada karyawan untuk filter ini" description="Coba ubah bulan atau divisi yang dipilih." />
                </TableCell>
              </TableRow>
            )}
            {!loading &&
              !error &&
              report?.employees.map((e) => (
                <TableRow key={e.employee_id}>
                  <TableCell className="font-mono text-sm">{e.employee_number}</TableCell>
                  <TableCell className="font-medium">{e.name}</TableCell>
                  <TableCell className="text-muted-foreground">{e.department_name || "-"}</TableCell>
                  <TableCell className="text-right">{e.working_days}</TableCell>
                  <TableCell className="text-right">{e.on_time}</TableCell>
                  <TableCell className="text-right">
                    <Count value={e.late_count} className="text-amber-600" />
                  </TableCell>
                  <TableCell className="text-right">{e.late_minutes}</TableCell>
                  <TableCell className="text-right">{e.izin}</TableCell>
                  <TableCell className="text-right">{e.sakit}</TableCell>
                  <TableCell className="text-right">{e.cuti}</TableCell>
                  <TableCell className="text-right">
                    <Count value={e.absent} className="text-destructive" />
                  </TableCell>
                </TableRow>
              ))}
          </TableBody>
        </Table>
      </div>

      <p className="mt-3 text-xs text-muted-foreground">
        &ldquo;Tidak Hadir&rdquo; adalah hari kerja yang sudah lewat tanpa absen dan tanpa keterangan izin/sakit/cuti
        (alpha). Keterangan diberikan di tab Harian. Rincian per tanggal ada di sheet &ldquo;Detail Harian&rdquo; pada
        file Excel.
      </p>
    </div>
  );
}

// --- Daily ---------------------------------------------------------------

type DerivedStatus = "ON_TIME" | "LATE" | "CHECKED_OUT" | LeaveType | "ABSENT";

const STATUS_LABELS: Record<DerivedStatus, string> = {
  ON_TIME: "Tepat Waktu",
  LATE: "Terlambat",
  CHECKED_OUT: "Selesai",
  IZIN: "Izin",
  SAKIT: "Sakit",
  CUTI: "Cuti",
  ABSENT: "Tidak Hadir",
};

const STATUS_VARIANTS: Record<DerivedStatus, "default" | "secondary" | "destructive" | "outline"> = {
  ON_TIME: "default",
  LATE: "destructive",
  CHECKED_OUT: "secondary",
  IZIN: "secondary",
  SAKIT: "secondary",
  CUTI: "secondary",
  ABSENT: "outline",
};

interface Row {
  employee: Employee;
  attendance: Attendance | null;
  leave: Leave | null;
  status: DerivedStatus;
}

function todayIso(): string {
  return new Date().toLocaleDateString("en-CA");
}

/**
 * The backend deliberately doesn't write an ABSENT row for an employee who
 * never checked in (see backend/README.md's Attendance section — ABSENT is
 * derived at report time, not a stored status). This view is that
 * derivation for a single day: cross-reference every active employee
 * against that day's attendance rows and recorded izin/sakit/cuti, entirely
 * client-side. The monthly tab does the equivalent server-side.
 *
 * It's also where HR gives a no-check-in day its keterangan: the device
 * can't know *why* someone didn't come in, so an "ABSENT" row offers a
 * one-click Izin/Sakit/Cuti label (a one-day leave record). Unlabelled days
 * stay alpha.
 */
function DailyReportView() {
  const { user } = useAuth();
  const writable = canWrite("leaves", user?.role);

  const [date, setDate] = useState(todayIso());
  const [departmentId, setDepartmentId] = useState(ALL_DEPARTMENTS);
  const [rows, setRows] = useState<Row[] | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [reloadKey, setReloadKey] = useState(0);

  const [labeling, setLabeling] = useState<Employee | null>(null);
  const [labelType, setLabelType] = useState<LeaveType>("IZIN");
  const [labelReason, setLabelReason] = useState("");
  const [saving, setSaving] = useState(false);
  const [cancelling, setCancelling] = useState<Leave | null>(null);

  const { options: departments } = useOptionsList<Department>("/departments");

  useEffect(() => {
    let cancelled = false;
    // The loading/error state must reset synchronously when the date or
    // department filter changes, not one render late.
    // eslint-disable-next-line react-hooks/set-state-in-effect
    setLoading(true);
    setError(null);

    async function load() {
      try {
        const [employeesRes, attendanceRes, leavesRes] = await Promise.all([
          api.get<ListResponse<Employee>>("/employees", {
            status: "ACTIVE",
            department_id: departmentId === ALL_DEPARTMENTS ? undefined : departmentId,
            page: 1,
            page_size: 200,
          }),
          api.get<ListResponse<Attendance>>("/attendance", {
            date_from: date,
            date_to: date,
            page: 1,
            page_size: 200,
          }),
          api.get<ListResponse<Leave>>("/leaves", {
            status: "ACTIVE",
            date_from: date,
            date_to: date,
            page: 1,
            page_size: 200,
          }),
        ]);
        if (cancelled) return;

        const attendanceByEmployee = new Map(attendanceRes.items.map((a) => [a.employee_id, a]));
        const leaveByEmployee = new Map(leavesRes.items.map((l) => [l.employee_id, l]));
        const built: Row[] = employeesRes.items.map((employee) => {
          const attendance = attendanceByEmployee.get(employee.id) ?? null;
          const leave = leaveByEmployee.get(employee.id) ?? null;
          const status: DerivedStatus =
            attendance?.status === "LATE"
              ? "LATE"
              : attendance?.status === "CHECKED_OUT"
                ? "CHECKED_OUT"
                : attendance?.status === "ON_TIME"
                  ? "ON_TIME"
                  : leave
                    ? leave.leave_type
                    : "ABSENT";
          return { employee, attendance, leave, status };
        });
        setRows(built);
      } catch {
        if (!cancelled) setError("Gagal memuat data laporan");
      } finally {
        if (!cancelled) setLoading(false);
      }
    }

    load();
    return () => {
      cancelled = true;
    };
  }, [date, departmentId, reloadKey]);

  function openLabel(employee: Employee) {
    setLabeling(employee);
    setLabelType("IZIN");
    setLabelReason("");
  }

  async function handleLabel(e: React.FormEvent) {
    e.preventDefault();
    if (!labeling) return;
    setSaving(true);
    try {
      await api.post("/leaves", {
        employee_id: labeling.id,
        leave_type: labelType,
        start_date: date,
        end_date: date,
        reason: labelReason.trim(),
      });
      toast.success(`${labeling.name} dicatat ${LEAVE_LABELS[labelType].toLowerCase()}`);
      setLabeling(null);
      setReloadKey((k) => k + 1);
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "Gagal menyimpan keterangan");
    } finally {
      setSaving(false);
    }
  }

  async function handleCancelLeave() {
    if (!cancelling) return;
    try {
      await api.delete(`/leaves/${cancelling.id}`);
      toast.success("Keterangan dibatalkan");
      setReloadKey((k) => k + 1);
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "Gagal membatalkan keterangan");
    }
  }

  const summary = rows?.reduce(
    (acc, row) => {
      acc[row.status] += 1;
      return acc;
    },
    { ON_TIME: 0, LATE: 0, CHECKED_OUT: 0, IZIN: 0, SAKIT: 0, CUTI: 0, ABSENT: 0 } as Record<DerivedStatus, number>,
  );
  const colSpan = writable ? 7 : 6;

  return (
    <div>
      <div className="mb-4 flex flex-wrap items-end gap-3 rounded-lg border bg-card p-4">
        <div className="space-y-1.5">
          <Label className="text-xs text-muted-foreground">Tanggal</Label>
          <Input type="date" value={date} onChange={(e) => setDate(e.target.value)} className="w-44" />
        </div>
        <DepartmentSelect departments={departments} value={departmentId} onChange={setDepartmentId} />
      </div>

      <div className="mb-6 grid gap-4 sm:grid-cols-3 xl:grid-cols-5">
        <StatTile label="Tepat Waktu" value={summary?.ON_TIME} icon={CheckCircle2} tone="green" loading={loading} />
        <StatTile label="Terlambat" value={summary?.LATE} icon={Clock} tone="amber" loading={loading} />
        <StatTile label="Selesai" value={summary?.CHECKED_OUT} icon={LogOut} tone="blue" loading={loading} />
        <StatTile
          label="Izin / Sakit / Cuti"
          value={summary ? summary.IZIN + summary.SAKIT + summary.CUTI : undefined}
          icon={ClipboardCheck}
          tone="violet"
          loading={loading}
        />
        <StatTile label="Tidak Hadir" value={summary?.ABSENT} icon={UserX} tone="rose" loading={loading} />
      </div>

      <div className="overflow-x-auto rounded-lg border bg-card">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>NIK</TableHead>
              <TableHead>Nama</TableHead>
              <TableHead>Divisi</TableHead>
              <TableHead>Check-In</TableHead>
              <TableHead>Check-Out</TableHead>
              <TableHead>Status</TableHead>
              {writable && <TableHead className="text-right">Keterangan</TableHead>}
            </TableRow>
          </TableHeader>
          <TableBody>
            {loading &&
              Array.from({ length: 6 }).map((_, i) => (
                <TableRow key={i}>
                  <TableCell colSpan={colSpan}>
                    <Skeleton className="h-5 w-full" />
                  </TableCell>
                </TableRow>
              ))}
            {!loading && error && (
              <TableRow>
                <TableCell colSpan={colSpan} className="py-8 text-center text-destructive">
                  {error}
                </TableCell>
              </TableRow>
            )}
            {!loading && !error && rows?.length === 0 && (
              <TableRow>
                <TableCell colSpan={colSpan}>
                  <EmptyState icon={Users} title="Tidak ada karyawan untuk filter ini" description="Coba ubah tanggal atau divisi yang dipilih." />
                </TableCell>
              </TableRow>
            )}
            {!loading &&
              !error &&
              rows?.map((row) => (
                <TableRow key={row.employee.id}>
                  <TableCell className="font-mono text-sm">{row.employee.employee_number}</TableCell>
                  <TableCell className="font-medium">{row.employee.name}</TableCell>
                  <TableCell className="text-muted-foreground">{row.employee.department_name ?? "-"}</TableCell>
                  <TableCell>
                    {row.attendance?.check_in_at
                      ? new Date(row.attendance.check_in_at).toLocaleTimeString("id-ID", {
                          hour: "2-digit",
                          minute: "2-digit",
                        })
                      : "-"}
                  </TableCell>
                  <TableCell>
                    {row.attendance?.check_out_at
                      ? new Date(row.attendance.check_out_at).toLocaleTimeString("id-ID", {
                          hour: "2-digit",
                          minute: "2-digit",
                        })
                      : "-"}
                  </TableCell>
                  <TableCell>
                    <Badge variant={STATUS_VARIANTS[row.status]}>{STATUS_LABELS[row.status]}</Badge>
                    {!row.attendance && row.leave?.reason && (
                      <span className="ml-2 text-xs text-muted-foreground">{row.leave.reason}</span>
                    )}
                  </TableCell>
                  {writable && (
                    <TableCell className="text-right">
                      {row.status === "ABSENT" && (
                        <Button size="sm" variant="outline" onClick={() => openLabel(row.employee)}>
                          Beri Keterangan
                        </Button>
                      )}
                      {!row.attendance && row.leave && (
                        <Button size="sm" variant="ghost" onClick={() => setCancelling(row.leave)}>
                          Batalkan
                        </Button>
                      )}
                    </TableCell>
                  )}
                </TableRow>
              ))}
          </TableBody>
        </Table>
      </div>

      <Dialog open={labeling !== null} onOpenChange={(open) => !open && setLabeling(null)}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>Keterangan Tidak Hadir</DialogTitle>
          </DialogHeader>
          <form onSubmit={handleLabel} className="space-y-4">
            <p className="text-sm text-muted-foreground">
              {labeling?.name} — {new Date(`${date}T00:00:00`).toLocaleDateString("id-ID", { dateStyle: "full" })}
            </p>
            <div className="space-y-2">
              <Label>Jenis</Label>
              <div className="flex gap-2">
                {(["IZIN", "SAKIT", "CUTI"] as const).map((t) => (
                  <Button
                    key={t}
                    type="button"
                    variant={labelType === t ? "default" : "outline"}
                    onClick={() => setLabelType(t)}
                    className="flex-1"
                  >
                    {LEAVE_LABELS[t]}
                  </Button>
                ))}
              </div>
              <p className="text-xs text-muted-foreground">
                {labelType === "CUTI"
                  ? "Cuti tidak memotong gaji, tetapi mengurangi jatah cuti tahunan (12 hari)."
                  : "Izin dan sakit memotong gaji, tetapi tidak mengurangi jatah cuti."}
              </p>
            </div>
            <div className="space-y-2">
              <Label htmlFor="label_reason">Keterangan</Label>
              <Textarea
                id="label_reason"
                value={labelReason}
                onChange={(e) => setLabelReason(e.target.value)}
                placeholder="mis. Demam, kabar via WA"
                maxLength={255}
              />
            </div>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => setLabeling(null)}>
                Batal
              </Button>
              <Button type="submit" disabled={saving}>
                {saving ? "Menyimpan..." : "Simpan"}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      <ConfirmDeleteDialog
        open={cancelling !== null}
        onOpenChange={(open) => !open && setCancelling(null)}
        title="Batalkan keterangan?"
        description={
          cancelling
            ? `${LEAVE_LABELS[cancelling.leave_type]} ${cancelling.employee_name} (${cancelling.start_date}${
                cancelling.end_date !== cancelling.start_date ? ` s/d ${cancelling.end_date}` : ""
              }) akan dibatalkan. Hari tersebut kembali dihitung Tidak Hadir.`
            : ""
        }
        onConfirm={handleCancelLeave}
      />
    </div>
  );
}

// --- Yearly ---------------------------------------------------------------

const YEARLY_CATEGORIES: { key: keyof YearlyReport["summary"]; label: string; icon: LucideIcon; tone: StatTone }[] = [
  { key: "izin", label: "Izin", icon: FileText, tone: "blue" },
  { key: "sakit", label: "Sakit", icon: Thermometer, tone: "violet" },
  { key: "absent", label: "Tidak Hadir", icon: UserX, tone: "rose" },
  { key: "cuti", label: "Cuti", icon: Palmtree, tone: "teal" },
  { key: "late", label: "Terlambat", icon: Clock, tone: "amber" },
];

function HeadcountCell({ h }: { h: Headcount }) {
  if (h.days === 0) return <span className="text-muted-foreground">-</span>;
  return (
    <span>
      {h.people} <span className="text-xs text-muted-foreground">org</span> · {h.days}{" "}
      <span className="text-xs text-muted-foreground">hr</span>
    </span>
  );
}

/**
 * The year-end recap HR used to build by hand: twelve monthly
 * spreadsheets merged into one, then tallied — how many people were izin,
 * sakit, alpha or cuti, and for how many days. The Excel export also carries
 * one day-grid sheet per month for checking.
 */
function YearlyReportView() {
  const thisYear = new Date().getFullYear();
  const [year, setYear] = useState(String(thisYear));
  const [departmentId, setDepartmentId] = useState(ALL_DEPARTMENTS);
  const [report, setReport] = useState<YearlyReport | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [exporting, setExporting] = useState(false);

  const { options: departments } = useOptionsList<Department>("/departments");
  const deptQuery = departmentId === ALL_DEPARTMENTS ? undefined : departmentId;
  const years = Array.from({ length: 5 }, (_, i) => String(thisYear - i));

  useEffect(() => {
    let cancelled = false;
    // eslint-disable-next-line react-hooks/set-state-in-effect
    setLoading(true);
    setError(null);

    api
      .get<YearlyReport>("/reports/yearly", { year, department_id: deptQuery })
      .then((res) => {
        if (!cancelled) setReport(res);
      })
      .catch((err: unknown) => {
        if (!cancelled) setError(err instanceof ApiError ? err.message : "Gagal memuat rekap tahunan");
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });

    return () => {
      cancelled = true;
    };
  }, [year, deptQuery]);

  async function handleExport() {
    setExporting(true);
    try {
      await downloadFile("/reports/yearly", { year, department_id: deptQuery, format: "xlsx" });
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "Gagal mengunduh file Excel");
    } finally {
      setExporting(false);
    }
  }

  return (
    <div>
      <div className="mb-4 flex flex-wrap items-end gap-3 rounded-lg border bg-card p-4">
        <div className="space-y-1.5">
          <Label className="text-xs text-muted-foreground">Tahun</Label>
          <Select items={Object.fromEntries(years.map((y) => [y, y]))} value={year} onValueChange={(v) => setYear(v ?? year)}>
            <SelectTrigger className="w-32">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {years.map((y) => (
                <SelectItem key={y} value={y}>
                  {y}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
        <DepartmentSelect departments={departments} value={departmentId} onChange={setDepartmentId} />
        <Button
          variant="outline"
          onClick={handleExport}
          disabled={exporting || loading || !report || report.employees.length === 0}
          className="ml-auto"
        >
          <Download className="size-4" />
          {exporting ? "Menyiapkan..." : "Export Excel"}
        </Button>
      </div>

      {error && <p className="mb-4 rounded-lg border bg-card p-4 text-center text-destructive">{error}</p>}

      <div className="mb-6 grid gap-4 sm:grid-cols-3 xl:grid-cols-5">
        {YEARLY_CATEGORIES.map((c) => (
          <StatTile
            key={c.key}
            label={`${c.label} (orang)`}
            value={report?.summary[c.key].people}
            hint={report ? `${report.summary[c.key].days} hari` : undefined}
            icon={c.icon}
            tone={c.tone}
            loading={loading}
          />
        ))}
      </div>

      <h3 className="mb-2 text-sm font-semibold">Per Bulan</h3>
      <div className="mb-6 overflow-x-auto rounded-lg border bg-card">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Bulan</TableHead>
              {YEARLY_CATEGORIES.map((c) => (
                <TableHead key={c.key} className="text-right">
                  {c.label}
                </TableHead>
              ))}
            </TableRow>
          </TableHeader>
          <TableBody>
            {loading &&
              Array.from({ length: 4 }).map((_, i) => (
                <TableRow key={i}>
                  <TableCell colSpan={6}>
                    <Skeleton className="h-5 w-full" />
                  </TableCell>
                </TableRow>
              ))}
            {!loading &&
              report?.months.map((m) => (
                <TableRow key={m.month}>
                  <TableCell className="font-medium">{MONTH_NAMES[m.month - 1]}</TableCell>
                  {YEARLY_CATEGORIES.map((c) => (
                    <TableCell key={c.key} className="text-right">
                      <HeadcountCell h={m[c.key]} />
                    </TableCell>
                  ))}
                </TableRow>
              ))}
          </TableBody>
        </Table>
      </div>

      <h3 className="mb-2 text-sm font-semibold">Per Karyawan</h3>
      <div className="overflow-x-auto rounded-lg border bg-card">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>NIK</TableHead>
              <TableHead>Nama</TableHead>
              <TableHead>Divisi</TableHead>
              <TableHead className="text-right">Hari Kerja</TableHead>
              <TableHead className="text-right">Hadir</TableHead>
              <TableHead className="text-right">Terlambat</TableHead>
              <TableHead className="text-right">Izin</TableHead>
              <TableHead className="text-right">Sakit</TableHead>
              <TableHead className="text-right">Cuti</TableHead>
              <TableHead className="text-right">Tidak Hadir</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {loading &&
              Array.from({ length: 6 }).map((_, i) => (
                <TableRow key={i}>
                  <TableCell colSpan={10}>
                    <Skeleton className="h-5 w-full" />
                  </TableCell>
                </TableRow>
              ))}
            {!loading && !error && report?.employees.length === 0 && (
              <TableRow>
                <TableCell colSpan={10}>
                  <EmptyState icon={Users} title="Tidak ada karyawan untuk filter ini" description="Coba ubah tahun atau divisi yang dipilih." />
                </TableCell>
              </TableRow>
            )}
            {!loading &&
              report?.employees.map((e) => (
                <TableRow key={e.employee_id}>
                  <TableCell className="font-mono text-sm">{e.employee_number}</TableCell>
                  <TableCell className="font-medium">{e.name}</TableCell>
                  <TableCell className="text-muted-foreground">{e.department_name || "-"}</TableCell>
                  <TableCell className="text-right">{e.working_days}</TableCell>
                  <TableCell className="text-right">{e.on_time + e.late_count}</TableCell>
                  <TableCell className="text-right">
                    <Count value={e.late_count} className="text-amber-600" />
                  </TableCell>
                  <TableCell className="text-right">{e.izin}</TableCell>
                  <TableCell className="text-right">{e.sakit}</TableCell>
                  <TableCell className="text-right">{e.cuti}</TableCell>
                  <TableCell className="text-right">
                    <Count value={e.absent} className="text-destructive" />
                  </TableCell>
                </TableRow>
              ))}
          </TableBody>
        </Table>
      </div>

      <p className="mt-3 text-xs text-muted-foreground">
        &ldquo;Orang&rdquo; = jumlah karyawan yang minimal sekali mengalaminya; &ldquo;hari&rdquo; = total harinya.
        Cuti yang sudah dijadwalkan untuk bulan mendatang ikut terhitung. File Excel berisi sheet Ringkasan, Per
        Karyawan, dan satu sheet per bulan (Jan–Des) untuk pengecekan.
      </p>
    </div>
  );
}
