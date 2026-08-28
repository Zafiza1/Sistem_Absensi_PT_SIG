"use client";

import { useState } from "react";

import { usePaginatedList } from "@/hooks/use-paginated-list";
import { useOptionsList } from "@/hooks/use-options-list";
import type { Attendance, AttendanceStatus, Employee } from "@/lib/types";

import { PageHeader } from "@/components/page-header";
import { DataTablePagination } from "@/components/data-table-pagination";
import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";

const ALL = "__all__";

const STATUS_LABELS: Record<AttendanceStatus, string> = {
  ON_TIME: "Tepat Waktu",
  LATE: "Terlambat",
  CHECKED_OUT: "Selesai",
  ABSENT: "Tidak Hadir",
  INCOMPLETE: "Belum Lengkap",
};

const STATUS_VARIANTS: Record<AttendanceStatus, "default" | "secondary" | "destructive" | "outline"> = {
  ON_TIME: "default",
  LATE: "destructive",
  CHECKED_OUT: "secondary",
  ABSENT: "destructive",
  INCOMPLETE: "outline",
};

function formatTime(value: string | null): string {
  if (!value) return "-";
  return new Date(value).toLocaleTimeString("id-ID", { hour: "2-digit", minute: "2-digit" });
}

export default function AttendancePage() {
  const [employeeId, setEmployeeId] = useState(ALL);
  const [status, setStatus] = useState(ALL);
  const [dateFrom, setDateFrom] = useState("");
  const [dateTo, setDateTo] = useState("");

  const { options: employees } = useOptionsList<Employee>("/employees");

  const { items, meta, setPage, loading, error } = usePaginatedList<Attendance>("/attendance", {
    employee_id: employeeId === ALL ? undefined : employeeId,
    status: status === ALL ? undefined : status,
    date_from: dateFrom || undefined,
    date_to: dateTo || undefined,
  });

  function resetFilters() {
    setEmployeeId(ALL);
    setStatus(ALL);
    setDateFrom("");
    setDateTo("");
  }

  return (
    <div>
      <PageHeader title="Riwayat Absensi" description="Riwayat check-in dan check-out seluruh karyawan" />

      <div className="mb-4 flex flex-wrap items-end gap-3 rounded-lg border bg-card p-4">
        <div className="space-y-1.5">
          <Label className="text-xs text-muted-foreground">Karyawan</Label>
          <Select
            items={{ [ALL]: "Semua karyawan", ...Object.fromEntries(employees.map((e) => [e.id, e.name])) }}
            value={employeeId}
            onValueChange={(v) => setEmployeeId(v ?? ALL)}
          >
            <SelectTrigger className="w-48">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value={ALL}>Semua karyawan</SelectItem>
              {employees.map((e) => (
                <SelectItem key={e.id} value={e.id}>
                  {e.name}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
        <div className="space-y-1.5">
          <Label className="text-xs text-muted-foreground">Status</Label>
          <Select
            items={{ [ALL]: "Semua status", ...STATUS_LABELS }}
            value={status}
            onValueChange={(v) => setStatus(v ?? ALL)}
          >
            <SelectTrigger className="w-40">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value={ALL}>Semua status</SelectItem>
              {Object.entries(STATUS_LABELS).map(([value, label]) => (
                <SelectItem key={value} value={value}>
                  {label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
        <div className="space-y-1.5">
          <Label className="text-xs text-muted-foreground">Dari Tanggal</Label>
          <Input type="date" value={dateFrom} onChange={(e) => setDateFrom(e.target.value)} className="w-40" />
        </div>
        <div className="space-y-1.5">
          <Label className="text-xs text-muted-foreground">Sampai Tanggal</Label>
          <Input type="date" value={dateTo} onChange={(e) => setDateTo(e.target.value)} className="w-40" />
        </div>
        <Button variant="ghost" onClick={resetFilters}>
          Reset
        </Button>
      </div>

      <div className="rounded-lg border bg-card">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Tanggal</TableHead>
              <TableHead>Karyawan</TableHead>
              <TableHead>Shift</TableHead>
              <TableHead>Check-In</TableHead>
              <TableHead>Check-Out</TableHead>
              <TableHead>Telat (menit)</TableHead>
              <TableHead>Status</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {loading &&
              Array.from({ length: 6 }).map((_, i) => (
                <TableRow key={i}>
                  <TableCell colSpan={7}>
                    <Skeleton className="h-5 w-full" />
                  </TableCell>
                </TableRow>
              ))}
            {!loading && error && (
              <TableRow>
                <TableCell colSpan={7} className="py-8 text-center text-destructive">
                  {error}
                </TableCell>
              </TableRow>
            )}
            {!loading && !error && items.length === 0 && (
              <TableRow>
                <TableCell colSpan={7} className="py-8 text-center text-muted-foreground">
                  Tidak ada data absensi untuk filter ini
                </TableCell>
              </TableRow>
            )}
            {!loading &&
              !error &&
              items.map((a) => (
                <TableRow key={a.id}>
                  <TableCell>{a.attendance_date}</TableCell>
                  <TableCell>
                    <p className="font-medium">{a.employee_name}</p>
                    <p className="text-xs text-muted-foreground">{a.employee_number}</p>
                  </TableCell>
                  <TableCell className="text-muted-foreground">{a.shift_name ?? "-"}</TableCell>
                  <TableCell>
                    {formatTime(a.check_in_at)}
                    {a.check_in_device_name && (
                      <span className="block text-xs text-muted-foreground">{a.check_in_device_name}</span>
                    )}
                  </TableCell>
                  <TableCell>
                    {formatTime(a.check_out_at)}
                    {a.check_out_device_name && (
                      <span className="block text-xs text-muted-foreground">{a.check_out_device_name}</span>
                    )}
                  </TableCell>
                  <TableCell>{a.late_minutes > 0 ? a.late_minutes : "-"}</TableCell>
                  <TableCell>
                    <Badge variant={STATUS_VARIANTS[a.status]}>{STATUS_LABELS[a.status]}</Badge>
                  </TableCell>
                </TableRow>
              ))}
          </TableBody>
        </Table>
        <DataTablePagination meta={meta} onPageChange={setPage} />
      </div>
    </div>
  );
}
