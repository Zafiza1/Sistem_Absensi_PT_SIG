"use client";

import { useState, useEffect, useCallback } from "react";
import { useRouter } from "next/navigation";
import { Plus, Search, Calendar, DollarSign, CheckCircle2, Lock, Play, Trash2, MoreHorizontal, Users, Calculator } from "lucide-react";
import { toast } from "sonner";

import { api, ApiError } from "@/lib/api-client";
import { useAuth } from "@/lib/auth-context";
import { canWrite } from "@/lib/permissions";
import type { PayrollPeriod, PayrollPeriodStatus } from "@/lib/types";

import { PageHeader } from "@/components/page-header";
import { EmptyState } from "@/components/empty-state";
import { Card } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { ConfirmDeleteDialog } from "@/components/confirm-delete-dialog";
import { RequireRole } from "@/components/require-role";

const STATUS_LABELS: Record<PayrollPeriodStatus, string> = {
  DRAFT: "Draft",
  PROCESSING: "Sedang Diproses",
  COMPLETED: "Selesai",
  LOCKED: "Terkunci",
};

const STATUS_VARIANTS: Record<PayrollPeriodStatus, "default" | "secondary" | "outline" | "destructive"> = {
  DRAFT: "secondary",
  PROCESSING: "outline",
  COMPLETED: "default",
  LOCKED: "destructive",
};

interface CreatePeriodForm {
  year: number;
  month: number;
  notes: string;
}

function formatCurrency(amount: number): string {
  return new Intl.NumberFormat("id-ID", {
    style: "currency",
    currency: "IDR",
    minimumFractionDigits: 0,
    maximumFractionDigits: 0,
  }).format(amount);
}

function formatDate(dateString: string): string {
  return new Date(dateString).toLocaleDateString("id-ID", {
    day: "numeric",
    month: "short",
    year: "numeric",
  });
}

function PayrollPageContent() {
  const { user } = useAuth();
  const router = useRouter();
  const writable = canWrite("payroll", user?.role);

  const [search, setSearch] = useState("");
  const [yearFilter, setYearFilter] = useState<number | undefined>(undefined);
  const [statusFilter, setStatusFilter] = useState<PayrollPeriodStatus | undefined>(undefined);
  const [periods, setPeriods] = useState<PayrollPeriod[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const [dialogOpen, setDialogOpen] = useState(false);
  const [form, setForm] = useState<CreatePeriodForm>({
    year: new Date().getFullYear(),
    month: new Date().getMonth() + 1,
    notes: "",
  });
  const [saving, setSaving] = useState(false);
  const [processingId, setProcessingId] = useState<string | null>(null);
  const [lockingId, setLockingId] = useState<string | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<PayrollPeriod | null>(null);

  // Load payroll periods
  const loadPeriods = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const params: Record<string, string | number> = {};
      if (yearFilter) params.year = yearFilter;
      if (statusFilter) params.status = statusFilter;

      const res = await api.get<{ items: PayrollPeriod[] }>("/payroll/periods", params);
      setPeriods(res.items);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Gagal memuat data payroll");
    } finally {
      setLoading(false);
    }
  }, [yearFilter, statusFilter]);

  // Initial load and filter changes
  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect -- loading state must flip on immediately when filters change
    loadPeriods();
  }, [loadPeriods]);

  function openCreateDialog() {
    setForm({
      year: new Date().getFullYear(),
      month: new Date().getMonth() + 1,
      notes: "",
    });
    setDialogOpen(true);
  }

  async function handleCreatePeriod() {
    setSaving(true);
    try {
      await api.post("/payroll/periods", form);
      toast.success("Periode payroll berhasil dibuat");
      setDialogOpen(false);
      loadPeriods();
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "Gagal membuat periode payroll");
    } finally {
      setSaving(false);
    }
  }

  async function handleProcessPeriod(period: PayrollPeriod) {
    setProcessingId(period.id);
    try {
      await api.post(`/payroll/periods/${period.id}/process`);
      toast.success("Payroll berhasil diproses");
      loadPeriods();
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "Gagal memproses payroll");
    } finally {
      setProcessingId(null);
    }
  }

  async function handleLockPeriod(period: PayrollPeriod) {
    setLockingId(period.id);
    try {
      await api.post(`/payroll/periods/${period.id}/lock`);
      toast.success("Periode payroll berhasil dikunci");
      loadPeriods();
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "Gagal mengunci periode payroll");
    } finally {
      setLockingId(null);
    }
  }

  async function handleDeletePeriod() {
    if (!deleteTarget) return;
    try {
      await api.delete(`/payroll/periods/${deleteTarget.id}`);
      toast.success("Periode payroll berhasil dihapus");
      loadPeriods();
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "Gagal menghapus periode payroll");
      throw err;
    }
  }

  const currentYear = new Date().getFullYear();
  const yearOptions = Array.from({ length: 5 }, (_, i) => currentYear - 2 + i);

  return (
    <div>
      <PageHeader
        title="Payroll"
        description="Kelola periode penggajian dan perhitungan gaji karyawan"
        action={
          <div className="flex gap-2">
            <Button variant="outline" onClick={() => router.push("/payroll/deduction-rules")}>
              <Calculator className="size-4" />
              Aturan Potongan
            </Button>
            {writable && (
              <Button onClick={openCreateDialog}>
                <Plus className="size-4" />
                Buat Periode Baru
              </Button>
            )}
          </div>
        }
      />

      <div className="mb-4 flex flex-wrap items-end gap-3 rounded-lg border bg-card p-4">
        <div className="relative max-w-sm flex-1">
          <Search className="pointer-events-none absolute top-1/2 left-2.5 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            placeholder="Cari periode..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="pl-8"
          />
        </div>
        <div className="space-y-1.5">
          <Label className="text-xs text-muted-foreground">Tahun</Label>
          <Select
            items={{ "": "Semua tahun", ...Object.fromEntries(yearOptions.map((y) => [y.toString(), y.toString()])) }}
            value={yearFilter?.toString() ?? ""}
            onValueChange={(v) => setYearFilter(v ? parseInt(v) : undefined)}
          >
            <SelectTrigger className="w-32">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="">Semua tahun</SelectItem>
              {yearOptions.map((y) => (
                <SelectItem key={y} value={y.toString()}>
                  {y}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
        <div className="space-y-1.5">
          <Label className="text-xs text-muted-foreground">Status</Label>
          <Select
            items={{
              "": "Semua status",
              DRAFT: "Draft",
              PROCESSING: "Sedang Diproses",
              COMPLETED: "Selesai",
              LOCKED: "Terkunci",
            }}
            value={statusFilter ?? ""}
            onValueChange={(v) => setStatusFilter((v as PayrollPeriodStatus) || undefined)}
          >
            <SelectTrigger className="w-40">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="">Semua status</SelectItem>
              <SelectItem value="DRAFT">Draft</SelectItem>
              <SelectItem value="PROCESSING">Sedang Diproses</SelectItem>
              <SelectItem value="COMPLETED">Selesai</SelectItem>
              <SelectItem value="LOCKED">Terkunci</SelectItem>
            </SelectContent>
          </Select>
        </div>
        <Button variant="outline" onClick={loadPeriods} className="ml-auto">
          Refresh
        </Button>
      </div>

      {/* Summary Cards */}
      <div className="mb-6 grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <Card className="flex-row items-center gap-3 p-4">
          <div className="flex size-10 shrink-0 items-center justify-center rounded-lg bg-primary/10 text-primary dark:bg-primary/20 dark:text-sky-400">
            <Calendar className="size-5" />
          </div>
          <div>
            <p className="text-xs text-muted-foreground">Total Periode</p>
            <p className="text-xl font-semibold tracking-tight">{periods.length}</p>
          </div>
        </Card>
        <Card className="flex-row items-center gap-3 p-4">
          <div className="flex size-10 shrink-0 items-center justify-center rounded-lg bg-emerald-50 text-emerald-600 dark:bg-emerald-500/10 dark:text-emerald-400">
            <Users className="size-5" />
          </div>
          <div>
            <p className="text-xs text-muted-foreground">Total Karyawan</p>
            <p className="text-xl font-semibold tracking-tight">
              {periods.reduce((sum, p) => sum + p.total_employees, 0)}
            </p>
          </div>
        </Card>
        <Card className="flex-row items-center gap-3 p-4">
          <div className="flex size-10 shrink-0 items-center justify-center rounded-lg bg-blue-50 text-blue-600 dark:bg-blue-500/10 dark:text-blue-400">
            <DollarSign className="size-5" />
          </div>
          <div>
            <p className="text-xs text-muted-foreground">Total Gaji Bruto</p>
            <p className="text-xl font-semibold tracking-tight">
              {formatCurrency(periods.reduce((sum, p) => sum + p.total_gross_pay, 0))}
            </p>
          </div>
        </Card>
        <Card className="flex-row items-center gap-3 p-4">
          <div className="flex size-10 shrink-0 items-center justify-center rounded-lg bg-green-50 text-green-600 dark:bg-green-500/10 dark:text-green-400">
            <CheckCircle2 className="size-5" />
          </div>
          <div>
            <p className="text-xs text-muted-foreground">Total Gaji Bersih</p>
            <p className="text-xl font-semibold tracking-tight">
              {formatCurrency(periods.reduce((sum, p) => sum + p.total_net_pay, 0))}
            </p>
          </div>
        </Card>
      </div>

      <div className="rounded-lg border bg-card">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Periode</TableHead>
              <TableHead>Status</TableHead>
              <TableHead className="text-right">Karyawan</TableHead>
              <TableHead className="text-right">Gaji Bruto</TableHead>
              <TableHead className="text-right">Gaji Bersih</TableHead>
              <TableHead className="text-right">Potongan</TableHead>
              <TableHead>Diproses</TableHead>
              {writable && <TableHead className="w-12" />}
            </TableRow>
          </TableHeader>
          <TableBody>
            {loading &&
              Array.from({ length: 5 }).map((_, i) => (
                <TableRow key={i}>
                  <TableCell colSpan={8}>
                    <Skeleton className="h-5 w-full" />
                  </TableCell>
                </TableRow>
              ))}
            {!loading && error && (
              <TableRow>
                <TableCell colSpan={8} className="py-8 text-center text-destructive">
                  {error}
                </TableCell>
              </TableRow>
            )}
            {!loading && !error && periods.length === 0 && (
              <TableRow>
                <TableCell colSpan={8}>
                  <EmptyState
                    icon={Calendar}
                    title="Tidak ada periode payroll ditemukan"
                    description="Mulai dengan membuat periode payroll baru."
                  />
                </TableCell>
              </TableRow>
            )}
            {!loading &&
              !error &&
              periods.map((period) => (
                <TableRow key={period.id}>
                  <TableCell>
                    <div>
                      <p className="font-medium">
                        {new Date(period.year, period.month - 1).toLocaleDateString("id-ID", {
                          month: "long",
                          year: "numeric",
                        })}
                      </p>
                      <p className="text-xs text-muted-foreground">
                        {formatDate(period.period_start)} - {formatDate(period.period_end)}
                      </p>
                    </div>
                  </TableCell>
                  <TableCell>
                    <Badge variant={STATUS_VARIANTS[period.status]}>{STATUS_LABELS[period.status]}</Badge>
                  </TableCell>
                  <TableCell className="text-right">{period.total_employees}</TableCell>
                  <TableCell className="text-right">{formatCurrency(period.total_gross_pay)}</TableCell>
                  <TableCell className="text-right">{formatCurrency(period.total_net_pay)}</TableCell>
                  <TableCell className="text-right text-destructive">
                    {formatCurrency(period.total_deductions)}
                  </TableCell>
                  <TableCell>
                    {period.processed_at ? (
                      <span className="text-xs text-muted-foreground">
                        {new Date(period.processed_at).toLocaleDateString("id-ID")}
                      </span>
                    ) : (
                      <span className="text-xs text-muted-foreground">-</span>
                    )}
                  </TableCell>
                  {writable && (
                    <TableCell>
                      <DropdownMenu>
                        <DropdownMenuTrigger render={<Button variant="ghost" size="icon" className="size-8" />}>
                          <MoreHorizontal className="size-4" />
                        </DropdownMenuTrigger>
                        <DropdownMenuContent align="end">
                          {period.status === "DRAFT" && (
                            <>
                              <DropdownMenuItem
                                onClick={() => handleProcessPeriod(period)}
                                disabled={processingId === period.id}
                              >
                                <Play className="mr-2 size-4" />
                                {processingId === period.id ? "Memproses..." : "Proses Payroll"}
                              </DropdownMenuItem>
                              <DropdownMenuItem
                                variant="destructive"
                                onClick={() => setDeleteTarget(period)}
                              >
                                <Trash2 className="mr-2 size-4" />
                                Hapus
                              </DropdownMenuItem>
                            </>
                          )}
                          {period.status === "COMPLETED" && (
                            <DropdownMenuItem
                              onClick={() => handleLockPeriod(period)}
                              disabled={lockingId === period.id}
                            >
                              <Lock className="mr-2 size-4" />
                              {lockingId === period.id ? "Mengunci..." : "Kunci Periode"}
                            </DropdownMenuItem>
                          )}
                          <DropdownMenuItem onClick={() => router.push(`/payroll/${period.id}/items`)}>
                            Lihat Detail
                          </DropdownMenuItem>
                        </DropdownMenuContent>
                      </DropdownMenu>
                    </TableCell>
                  )}
                </TableRow>
              ))}
          </TableBody>
        </Table>
      </div>

      {/* Create Period Dialog */}
      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>Buat Periode Payroll Baru</DialogTitle>
          </DialogHeader>
          <div className="space-y-4">
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label htmlFor="year">Tahun</Label>
                <Select
                  items={Object.fromEntries(yearOptions.map((y) => [y.toString(), y.toString()]))}
                  value={form.year.toString()}
                  onValueChange={(v) => setForm((f) => (v ? { ...f, year: parseInt(v) } : f))}
                >
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {yearOptions.map((y) => (
                      <SelectItem key={y} value={y.toString()}>
                        {y}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
              <div className="space-y-2">
                <Label htmlFor="month">Bulan</Label>
                <Select
                  items={Object.fromEntries(
                    Array.from({ length: 12 }, (_, i) => [
                      (i + 1).toString(),
                      new Date(0, i).toLocaleDateString("id-ID", { month: "long" }),
                    ]),
                  )}
                  value={form.month.toString()}
                  onValueChange={(v) => setForm((f) => (v ? { ...f, month: parseInt(v) } : f))}
                >
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {Array.from({ length: 12 }, (_, i) => (
                      <SelectItem key={i + 1} value={(i + 1).toString()}>
                        {new Date(0, i).toLocaleDateString("id-ID", { month: "long" })}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
            </div>
            <div className="space-y-2">
              <Label htmlFor="notes">Catatan (opsional)</Label>
              <Input
                id="notes"
                value={form.notes}
                onChange={(e) => setForm((f) => ({ ...f, notes: e.target.value }))}
                placeholder="Catatan untuk periode ini"
              />
            </div>
          </div>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => setDialogOpen(false)}>
              Batal
            </Button>
            <Button type="button" onClick={handleCreatePeriod} disabled={saving}>
              {saving ? "Menyimpan..." : "Buat Periode"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <ConfirmDeleteDialog
        open={!!deleteTarget}
        onOpenChange={(open) => !open && setDeleteTarget(null)}
        title="Hapus periode payroll?"
        description={`Periode "${new Date(deleteTarget?.year || 0, (deleteTarget?.month || 1) - 1).toLocaleDateString("id-ID", { month: "long", year: "numeric" })}" akan dihapus. Tindakan ini tidak dapat dibatalkan.`}
        onConfirm={handleDeletePeriod}
      />
    </div>
  );
}

export default function PayrollPage() {
  return (
    <RequireRole roles={["SUPER_ADMIN", "ADMIN", "HR"]}>
      <PayrollPageContent />
    </RequireRole>
  );
}