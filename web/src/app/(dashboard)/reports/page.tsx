"use client";

import { useEffect, useState } from "react";

import { api } from "@/lib/api-client";
import { useOptionsList } from "@/hooks/use-options-list";
import type { Attendance, Department, Employee, ListResponse } from "@/lib/types";

import { PageHeader } from "@/components/page-header";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";

const ALL_DEPARTMENTS = "__all__";

type DerivedStatus = "ON_TIME" | "LATE" | "CHECKED_OUT" | "ABSENT";

const STATUS_LABELS: Record<DerivedStatus, string> = {
  ON_TIME: "Tepat Waktu",
  LATE: "Terlambat",
  CHECKED_OUT: "Selesai",
  ABSENT: "Tidak Hadir",
};

const STATUS_VARIANTS: Record<DerivedStatus, "default" | "secondary" | "destructive" | "outline"> = {
  ON_TIME: "default",
  LATE: "destructive",
  CHECKED_OUT: "secondary",
  ABSENT: "outline",
};

interface Row {
  employee: Employee;
  attendance: Attendance | null;
  status: DerivedStatus;
}

function todayIso(): string {
  return new Date().toLocaleDateString("en-CA");
}

/**
 * The backend deliberately doesn't write an ABSENT row for an employee who
 * never checked in (see backend/README.md's Attendance section — ABSENT is
 * "derived ... planned for Phase 6's reporting", not a stored status). This
 * page is that derivation: cross-reference every active employee against
 * that day's attendance rows, entirely client-side, rather than adding a
 * new backend endpoint out of scope for this pass.
 */
export default function ReportsPage() {
  const [date, setDate] = useState(todayIso());
  const [departmentId, setDepartmentId] = useState(ALL_DEPARTMENTS);
  const [rows, setRows] = useState<Row[] | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

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
        const [employeesRes, attendanceRes] = await Promise.all([
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
        ]);
        if (cancelled) return;

        const attendanceByEmployee = new Map(attendanceRes.items.map((a) => [a.employee_id, a]));
        const built: Row[] = employeesRes.items.map((employee) => {
          const attendance = attendanceByEmployee.get(employee.id) ?? null;
          const status: DerivedStatus =
            attendance?.status === "LATE"
              ? "LATE"
              : attendance?.status === "CHECKED_OUT"
                ? "CHECKED_OUT"
                : attendance?.status === "ON_TIME"
                  ? "ON_TIME"
                  : "ABSENT";
          return { employee, attendance, status };
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
  }, [date, departmentId]);

  const summary = rows?.reduce(
    (acc, row) => {
      acc[row.status] += 1;
      return acc;
    },
    { ON_TIME: 0, LATE: 0, CHECKED_OUT: 0, ABSENT: 0 } as Record<DerivedStatus, number>,
  );

  return (
    <div>
      <PageHeader title="Laporan Kehadiran" description="Ringkasan kehadiran karyawan per hari" />

      <div className="mb-4 flex flex-wrap items-end gap-3 rounded-lg border bg-card p-4">
        <div className="space-y-1.5">
          <Label className="text-xs text-muted-foreground">Tanggal</Label>
          <Input type="date" value={date} onChange={(e) => setDate(e.target.value)} className="w-44" />
        </div>
        <div className="space-y-1.5">
          <Label className="text-xs text-muted-foreground">Departemen</Label>
          <Select
            items={{ [ALL_DEPARTMENTS]: "Semua departemen", ...Object.fromEntries(departments.map((d) => [d.id, d.name])) }}
            value={departmentId}
            onValueChange={(v) => setDepartmentId(v ?? ALL_DEPARTMENTS)}
          >
            <SelectTrigger className="w-48">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value={ALL_DEPARTMENTS}>Semua departemen</SelectItem>
              {departments.map((d) => (
                <SelectItem key={d.id} value={d.id}>
                  {d.name}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      </div>

      <div className="mb-6 grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        {(["ON_TIME", "LATE", "CHECKED_OUT", "ABSENT"] as const).map((key) => (
          <Card key={key}>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm font-medium text-muted-foreground">{STATUS_LABELS[key]}</CardTitle>
            </CardHeader>
            <CardContent>
              {loading ? <Skeleton className="h-8 w-12" /> : <div className="text-2xl font-bold">{summary?.[key] ?? 0}</div>}
            </CardContent>
          </Card>
        ))}
      </div>

      <div className="rounded-lg border bg-card">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>NIK</TableHead>
              <TableHead>Nama</TableHead>
              <TableHead>Departemen</TableHead>
              <TableHead>Check-In</TableHead>
              <TableHead>Check-Out</TableHead>
              <TableHead>Status</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {loading &&
              Array.from({ length: 6 }).map((_, i) => (
                <TableRow key={i}>
                  <TableCell colSpan={6}>
                    <Skeleton className="h-5 w-full" />
                  </TableCell>
                </TableRow>
              ))}
            {!loading && error && (
              <TableRow>
                <TableCell colSpan={6} className="py-8 text-center text-destructive">
                  {error}
                </TableCell>
              </TableRow>
            )}
            {!loading && !error && rows?.length === 0 && (
              <TableRow>
                <TableCell colSpan={6} className="py-8 text-center text-muted-foreground">
                  Tidak ada karyawan untuk filter ini
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
                  </TableCell>
                </TableRow>
              ))}
          </TableBody>
        </Table>
      </div>
    </div>
  );
}
