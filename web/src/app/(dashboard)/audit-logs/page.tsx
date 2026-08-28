"use client";

import { useState } from "react";

import { usePaginatedList } from "@/hooks/use-paginated-list";
import { useOptionsList } from "@/hooks/use-options-list";
import type { AuditAction, AuditLogEntry, DashboardUser } from "@/lib/types";

import { RequireRole } from "@/components/require-role";
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

const ACTION_LABELS: Record<AuditAction, string> = {
  CREATE: "Buat",
  UPDATE: "Ubah",
  DELETE: "Hapus",
  LOGIN: "Login",
  LOGIN_FAILED: "Login Gagal",
};

const ACTION_VARIANTS: Record<AuditAction, "default" | "secondary" | "destructive" | "outline"> = {
  CREATE: "default",
  UPDATE: "outline",
  DELETE: "destructive",
  LOGIN: "secondary",
  LOGIN_FAILED: "destructive",
};

function formatDateTime(value: string): string {
  return new Date(value).toLocaleString("id-ID", { dateStyle: "medium", timeStyle: "medium" });
}

function AuditLogsPageContent() {
  const [actorId, setActorId] = useState(ALL);
  const [action, setAction] = useState(ALL);
  const [dateFrom, setDateFrom] = useState("");
  const [dateTo, setDateTo] = useState("");

  const { options: users } = useOptionsList<DashboardUser>("/users");

  const { items, meta, setPage, loading, error } = usePaginatedList<AuditLogEntry>("/audit-logs", {
    actor_id: actorId === ALL ? undefined : actorId,
    action: action === ALL ? undefined : action,
    date_from: dateFrom || undefined,
    date_to: dateTo || undefined,
  });

  function resetFilters() {
    setActorId(ALL);
    setAction(ALL);
    setDateFrom("");
    setDateTo("");
  }

  return (
    <div>
      <PageHeader title="Audit Log" description="Riwayat aktivitas login dan perubahan data di sistem" />

      <div className="mb-4 flex flex-wrap items-end gap-3 rounded-lg border bg-card p-4">
        <div className="space-y-1.5">
          <Label className="text-xs text-muted-foreground">Pengguna</Label>
          <Select
            items={{ [ALL]: "Semua pengguna", ...Object.fromEntries(users.map((u) => [u.id, u.name])) }}
            value={actorId}
            onValueChange={(v) => setActorId(v ?? ALL)}
          >
            <SelectTrigger className="w-48">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value={ALL}>Semua pengguna</SelectItem>
              {users.map((u) => (
                <SelectItem key={u.id} value={u.id}>
                  {u.name}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
        <div className="space-y-1.5">
          <Label className="text-xs text-muted-foreground">Aksi</Label>
          <Select
            items={{ [ALL]: "Semua aksi", ...ACTION_LABELS }}
            value={action}
            onValueChange={(v) => setAction(v ?? ALL)}
          >
            <SelectTrigger className="w-40">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value={ALL}>Semua aksi</SelectItem>
              {Object.entries(ACTION_LABELS).map(([value, label]) => (
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
              <TableHead>Waktu</TableHead>
              <TableHead>Pengguna</TableHead>
              <TableHead>Aksi</TableHead>
              <TableHead>Deskripsi</TableHead>
              <TableHead>IP</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {loading &&
              Array.from({ length: 6 }).map((_, i) => (
                <TableRow key={i}>
                  <TableCell colSpan={5}>
                    <Skeleton className="h-5 w-full" />
                  </TableCell>
                </TableRow>
              ))}
            {!loading && error && (
              <TableRow>
                <TableCell colSpan={5} className="py-8 text-center text-destructive">
                  {error}
                </TableCell>
              </TableRow>
            )}
            {!loading && !error && items.length === 0 && (
              <TableRow>
                <TableCell colSpan={5} className="py-8 text-center text-muted-foreground">
                  Tidak ada aktivitas untuk filter ini
                </TableCell>
              </TableRow>
            )}
            {!loading &&
              !error &&
              items.map((entry) => (
                <TableRow key={entry.id}>
                  <TableCell className="whitespace-nowrap text-sm">{formatDateTime(entry.created_at)}</TableCell>
                  <TableCell>
                    <p className="font-medium">{entry.actor_name}</p>
                    {entry.actor_role && <p className="text-xs text-muted-foreground">{entry.actor_role}</p>}
                  </TableCell>
                  <TableCell>
                    <Badge variant={ACTION_VARIANTS[entry.action]}>{ACTION_LABELS[entry.action]}</Badge>
                  </TableCell>
                  <TableCell className="max-w-md text-muted-foreground">{entry.description}</TableCell>
                  <TableCell className="font-mono text-xs text-muted-foreground">{entry.ip_address || "-"}</TableCell>
                </TableRow>
              ))}
          </TableBody>
        </Table>
        <DataTablePagination meta={meta} onPageChange={setPage} />
      </div>
    </div>
  );
}

export default function AuditLogsPage() {
  return (
    <RequireRole roles={["SUPER_ADMIN"]}>
      <AuditLogsPageContent />
    </RequireRole>
  );
}
