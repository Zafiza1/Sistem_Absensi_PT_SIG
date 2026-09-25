"use client";

import { useState, useEffect, useCallback } from "react";
import { ArrowLeft, Search, DollarSign, Calculator, UserMinus, CheckCircle2, CreditCard, Pencil } from "lucide-react";
import { useParams, useRouter } from "next/navigation";
import { toast } from "sonner";

import { api, ApiError } from "@/lib/api-client";
import { useAuth } from "@/lib/auth-context";
import { canWrite } from "@/lib/permissions";
import type { PayrollItem, PaymentStatus, PayrollPeriod } from "@/lib/types";

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
import { RequireRole } from "@/components/require-role";

const PAYMENT_STATUS_LABELS: Record<PaymentStatus, string> = {
  PENDING: "Belum Dibayar",
  PAID: "Sudah Dibayar",
  FAILED: "Gagal",
};

const PAYMENT_STATUS_VARIANTS: Record<PaymentStatus, "default" | "secondary" | "destructive"> = {
  PENDING: "secondary",
  PAID: "default",
  FAILED: "destructive",
};

interface MarkAsPaidForm {
  payment_date: string;
  payment_method: string;
  payment_reference: string;
}

interface EditItemForm {
  allowance: number;
  bonus: number;
  other_earnings: number;
  tax_deduction: number;
  insurance_deduction: number;
  other_deductions: number;
  notes: string;
}

const EMPTY_EDIT_ITEM_FORM: EditItemForm = {
  allowance: 0,
  bonus: 0,
  other_earnings: 0,
  tax_deduction: 0,
  insurance_deduction: 0,
  other_deductions: 0,
  notes: "",
};

function toEditItemForm(item: PayrollItem): EditItemForm {
  return {
    allowance: item.allowance,
    bonus: item.bonus,
    other_earnings: item.other_earnings,
    tax_deduction: item.tax_deduction,
    insurance_deduction: item.insurance_deduction,
    other_deductions: item.other_deductions,
    notes: item.notes ?? "",
  };
}

function formatCurrency(amount: number): string {
  return new Intl.NumberFormat("id-ID", {
    style: "currency",
    currency: "IDR",
    minimumFractionDigits: 0,
    maximumFractionDigits: 0,
  }).format(amount);
}

function PayrollItemsPageContent() {
  const { id } = useParams<{ id: string }>();
  const router = useRouter();
  const { user } = useAuth();
  const writable = canWrite("payroll", user?.role);

  const [period, setPeriod] = useState<PayrollPeriod | null>(null);
  const [items, setItems] = useState<PayrollItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [search, setSearch] = useState("");

  const [markPaidDialogOpen, setMarkPaidDialogOpen] = useState(false);
  const [markPaidForm, setMarkPaidForm] = useState<MarkAsPaidForm>({
    payment_date: new Date().toISOString().split("T")[0],
    payment_method: "Bank Transfer",
    payment_reference: "",
  });
  const [markingAsPaid, setMarkingAsPaid] = useState(false);

  const [editItemDialogOpen, setEditItemDialogOpen] = useState(false);
  const [editingItem, setEditingItem] = useState<PayrollItem | null>(null);
  const [editItemForm, setEditItemForm] = useState<EditItemForm>(EMPTY_EDIT_ITEM_FORM);
  const [savingItem, setSavingItem] = useState(false);

  // Load period details and items
  const loadData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const [periodRes, itemsRes] = await Promise.all([
        api.get<PayrollPeriod>(`/payroll/periods/${id}`),
        api.get<{ items: PayrollItem[] }>(`/payroll/periods/${id}/items`),
      ]);
      setPeriod(periodRes);
      setItems(itemsRes.items);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Gagal memuat data payroll");
    } finally {
      setLoading(false);
    }
  }, [id]);

  useEffect(() => {
    if (id) {
      // eslint-disable-next-line react-hooks/set-state-in-effect -- loading state must flip on immediately when id changes
      loadData();
    }
  }, [id, loadData]);

  async function handleMarkAsPaid() {
    setMarkingAsPaid(true);
    try {
      await api.post(`/payroll/periods/${id}/mark-paid`, markPaidForm);
      toast.success("Payroll berhasil ditandai sebagai dibayar");
      setMarkPaidDialogOpen(false);
      loadData();
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "Gagal menandai sebagai dibayar");
    } finally {
      setMarkingAsPaid(false);
    }
  }

  function openEditItemDialog(item: PayrollItem) {
    setEditingItem(item);
    setEditItemForm(toEditItemForm(item));
    setEditItemDialogOpen(true);
  }

  async function handleSaveItem() {
    if (!editingItem) return;
    setSavingItem(true);
    try {
      await api.put(`/payroll/items/${editingItem.id}`, editItemForm);
      toast.success("Item payroll berhasil diperbarui");
      setEditItemDialogOpen(false);
      loadData();
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "Gagal memperbarui item payroll");
    } finally {
      setSavingItem(false);
    }
  }

  const filteredItems = items.filter(
    (item) =>
      item.employee_name.toLowerCase().includes(search.toLowerCase()) ||
      item.employee_number.toLowerCase().includes(search.toLowerCase()) ||
      item.department_name.toLowerCase().includes(search.toLowerCase()),
  );

  const summary = items.reduce(
    (acc, item) => {
      acc.total_earnings += item.total_earnings;
      acc.total_deductions += item.total_deductions;
      acc.total_net_pay += item.net_pay;
      acc.total_employees += 1;
      acc.total_late_days += item.late_days;
      acc.total_absent_days += item.absent_days;
      return acc;
    },
    {
      total_earnings: 0,
      total_deductions: 0,
      total_net_pay: 0,
      total_employees: 0,
      total_late_days: 0,
      total_absent_days: 0,
    },
  );

  return (
    <div>
      <PageHeader
        title="Detail Payroll"
        description="Rincian gaji karyawan untuk periode ini"
        action={
          <Button variant="outline" onClick={() => router.back()}>
            <ArrowLeft className="size-4" />
            Kembali
          </Button>
        }
      />

      {period && (
        <div className="mb-6 rounded-lg border bg-card p-4">
          <div className="mb-4">
            <h3 className="text-lg font-semibold">
              {new Date(period.year, period.month - 1).toLocaleDateString("id-ID", {
                month: "long",
                year: "numeric",
              })}
            </h3>
            <p className="text-sm text-muted-foreground">
              {new Date(period.period_start).toLocaleDateString("id-ID")} -{" "}
              {new Date(period.period_end).toLocaleDateString("id-ID")}
            </p>
          </div>
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
            <Card className="flex-row items-center gap-3 p-4">
              <div className="flex size-10 shrink-0 items-center justify-center rounded-lg bg-blue-50 text-blue-600 dark:bg-blue-500/10 dark:text-blue-400">
                <Calculator className="size-5" />
              </div>
              <div>
                <p className="text-xs text-muted-foreground">Total Pendapatan</p>
                <p className="text-lg font-semibold">{formatCurrency(summary.total_earnings)}</p>
              </div>
            </Card>
            <Card className="flex-row items-center gap-3 p-4">
              <div className="flex size-10 shrink-0 items-center justify-center rounded-lg bg-rose-50 text-rose-600 dark:bg-rose-500/10 dark:text-rose-400">
                <UserMinus className="size-5" />
              </div>
              <div>
                <p className="text-xs text-muted-foreground">Total Potongan</p>
                <p className="text-lg font-semibold">{formatCurrency(summary.total_deductions)}</p>
              </div>
            </Card>
            <Card className="flex-row items-center gap-3 p-4">
              <div className="flex size-10 shrink-0 items-center justify-center rounded-lg bg-green-50 text-green-600 dark:bg-green-500/10 dark:text-green-400">
                <DollarSign className="size-5" />
              </div>
              <div>
                <p className="text-xs text-muted-foreground">Total Gaji Bersih</p>
                <p className="text-lg font-semibold">{formatCurrency(summary.total_net_pay)}</p>
              </div>
            </Card>
            <Card className="flex-row items-center gap-3 p-4">
              <div className="flex size-10 shrink-0 items-center justify-center rounded-lg bg-primary/10 text-primary dark:bg-primary/20 dark:text-sky-400">
                <CheckCircle2 className="size-5" />
              </div>
              <div>
                <p className="text-xs text-muted-foreground">Status Pembayaran</p>
                <Badge variant={PAYMENT_STATUS_VARIANTS[period.status as PaymentStatus] || "secondary"}>
                  {PAYMENT_STATUS_LABELS[period.status as PaymentStatus] || period.status}
                </Badge>
              </div>
            </Card>
          </div>
        </div>
      )}

      <div className="mb-4 flex items-center gap-3">
        <div className="relative max-w-sm flex-1">
          <Search className="pointer-events-none absolute top-1/2 left-2.5 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            placeholder="Cari karyawan..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="pl-8"
          />
        </div>
        {writable && period?.status === "COMPLETED" && (
          <Button onClick={() => setMarkPaidDialogOpen(true)}>
            <CreditCard className="size-4" />
            Tandai Sudah Dibayar
          </Button>
        )}
      </div>

      <div className="rounded-lg border bg-card">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>NIK</TableHead>
              <TableHead>Nama</TableHead>
              <TableHead>Divisi</TableHead>
              <TableHead>Jabatan</TableHead>
              <TableHead className="text-right">Hari Kerja</TableHead>
              <TableHead className="text-right">Terlambat</TableHead>
              <TableHead className="text-right">Absen</TableHead>
              <TableHead className="text-right">Gaji Pokok</TableHead>
              <TableHead className="text-right">Pendapatan</TableHead>
              <TableHead className="text-right">Potongan</TableHead>
              <TableHead className="text-right">Gaji Bersih</TableHead>
              <TableHead>Status</TableHead>
              {writable && period?.status !== "LOCKED" && <TableHead className="w-12" />}
            </TableRow>
          </TableHeader>
          <TableBody>
            {loading &&
              Array.from({ length: 5 }).map((_, i) => (
                <TableRow key={i}>
                  <TableCell colSpan={13}>
                    <Skeleton className="h-5 w-full" />
                  </TableCell>
                </TableRow>
              ))}
            {!loading && error && (
              <TableRow>
                <TableCell colSpan={13} className="py-8 text-center text-destructive">
                  {error}
                </TableCell>
              </TableRow>
            )}
            {!loading && !error && filteredItems.length === 0 && (
              <TableRow>
                <TableCell colSpan={13}>
                  <EmptyState
                    icon={Calculator}
                    title="Tidak ada data payroll ditemukan"
                    description={search ? "Coba kata kunci pencarian lain." : "Belum ada data payroll untuk periode ini."}
                  />
                </TableCell>
              </TableRow>
            )}
            {!loading &&
              !error &&
              filteredItems.map((item) => (
                <TableRow key={item.id}>
                  <TableCell className="font-mono text-sm">{item.employee_number}</TableCell>
                  <TableCell className="font-medium">{item.employee_name}</TableCell>
                  <TableCell className="text-muted-foreground">{item.department_name}</TableCell>
                  <TableCell className="text-muted-foreground">{item.position_name}</TableCell>
                  <TableCell className="text-right">{item.working_days}</TableCell>
                  <TableCell className="text-right">
                    {item.late_days > 0 ? (
                      <span className="font-medium text-amber-600">{item.late_days}</span>
                    ) : (
                      0
                    )}
                  </TableCell>
                  <TableCell className="text-right">
                    {item.absent_days > 0 ? (
                      <span className="font-medium text-destructive">{item.absent_days}</span>
                    ) : (
                      0
                    )}
                  </TableCell>
                  <TableCell className="text-right">{formatCurrency(item.base_salary)}</TableCell>
                  <TableCell className="text-right text-green-600">{formatCurrency(item.total_earnings)}</TableCell>
                  <TableCell className="text-right text-destructive">{formatCurrency(item.total_deductions)}</TableCell>
                  <TableCell className="text-right font-semibold">{formatCurrency(item.net_pay)}</TableCell>
                  <TableCell>
                    <Badge variant={PAYMENT_STATUS_VARIANTS[item.payment_status]}>
                      {PAYMENT_STATUS_LABELS[item.payment_status]}
                    </Badge>
                  </TableCell>
                  {writable && period?.status !== "LOCKED" && (
                    <TableCell>
                      <Button
                        variant="ghost"
                        size="icon"
                        className="size-8"
                        onClick={() => openEditItemDialog(item)}
                      >
                        <Pencil className="size-4" />
                      </Button>
                    </TableCell>
                  )}
                </TableRow>
              ))}
          </TableBody>
        </Table>
      </div>

      {/* Mark as Paid Dialog */}
      <Dialog open={markPaidDialogOpen} onOpenChange={setMarkPaidDialogOpen}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>Tandai Payroll Sudah Dibayar</DialogTitle>
          </DialogHeader>
          <div className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="payment_date">Tanggal Pembayaran</Label>
              <Input
                id="payment_date"
                type="date"
                value={markPaidForm.payment_date}
                onChange={(e) => setMarkPaidForm((f) => ({ ...f, payment_date: e.target.value }))}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="payment_method">Metode Pembayaran</Label>
              <Select
                items={{
                  "Bank Transfer": "Bank Transfer",
                  "Cash": "Tunai",
                  "Check": "Cek",
                  "Other": "Lainnya",
                }}
                value={markPaidForm.payment_method}
                onValueChange={(v) => setMarkPaidForm((f) => ({ ...f, payment_method: v || "Bank Transfer" }))}
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="Bank Transfer">Bank Transfer</SelectItem>
                  <SelectItem value="Cash">Tunai</SelectItem>
                  <SelectItem value="Check">Cek</SelectItem>
                  <SelectItem value="Other">Lainnya</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-2">
              <Label htmlFor="payment_reference">Referensi Pembayaran (opsional)</Label>
              <Input
                id="payment_reference"
                value={markPaidForm.payment_reference}
                onChange={(e) => setMarkPaidForm((f) => ({ ...f, payment_reference: e.target.value }))}
                placeholder="Nomor referensi atau keterangan"
              />
            </div>
          </div>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => setMarkPaidDialogOpen(false)}>
              Batal
            </Button>
            <Button type="button" onClick={handleMarkAsPaid} disabled={markingAsPaid}>
              {markingAsPaid ? "Memproses..." : "Tandai Sudah Dibayar"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Edit Payroll Item Dialog */}
      <Dialog open={editItemDialogOpen} onOpenChange={setEditItemDialogOpen}>
        <DialogContent className="max-h-[90vh] overflow-y-auto sm:max-w-md">
          <DialogHeader>
            <DialogTitle>Edit Item Payroll</DialogTitle>
          </DialogHeader>
          {editingItem && (
            <div className="space-y-4">
              <p className="text-sm text-muted-foreground">
                {editingItem.employee_name} ({editingItem.employee_number})
              </p>
              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-2">
                  <Label htmlFor="allowance">Tunjangan</Label>
                  <Input
                    id="allowance"
                    type="number"
                    value={editItemForm.allowance}
                    onChange={(e) => setEditItemForm((f) => ({ ...f, allowance: parseInt(e.target.value) || 0 }))}
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="bonus">Bonus</Label>
                  <Input
                    id="bonus"
                    type="number"
                    value={editItemForm.bonus}
                    onChange={(e) => setEditItemForm((f) => ({ ...f, bonus: parseInt(e.target.value) || 0 }))}
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="other_earnings">Pendapatan Lain</Label>
                  <Input
                    id="other_earnings"
                    type="number"
                    value={editItemForm.other_earnings}
                    onChange={(e) => setEditItemForm((f) => ({ ...f, other_earnings: parseInt(e.target.value) || 0 }))}
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="tax_deduction">Potongan Pajak</Label>
                  <Input
                    id="tax_deduction"
                    type="number"
                    value={editItemForm.tax_deduction}
                    onChange={(e) => setEditItemForm((f) => ({ ...f, tax_deduction: parseInt(e.target.value) || 0 }))}
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="insurance_deduction">Potongan Asuransi</Label>
                  <Input
                    id="insurance_deduction"
                    type="number"
                    value={editItemForm.insurance_deduction}
                    onChange={(e) => setEditItemForm((f) => ({ ...f, insurance_deduction: parseInt(e.target.value) || 0 }))}
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="other_deductions">Potongan Lain</Label>
                  <Input
                    id="other_deductions"
                    type="number"
                    value={editItemForm.other_deductions}
                    onChange={(e) => setEditItemForm((f) => ({ ...f, other_deductions: parseInt(e.target.value) || 0 }))}
                  />
                </div>
              </div>
              <div className="space-y-2">
                <Label htmlFor="item_notes">Catatan (opsional)</Label>
                <Input
                  id="item_notes"
                  value={editItemForm.notes}
                  onChange={(e) => setEditItemForm((f) => ({ ...f, notes: e.target.value }))}
                  placeholder="Catatan untuk item payroll ini"
                />
              </div>
            </div>
          )}
          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => setEditItemDialogOpen(false)}>
              Batal
            </Button>
            <Button type="button" onClick={handleSaveItem} disabled={savingItem}>
              {savingItem ? "Menyimpan..." : "Simpan"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}

export default function PayrollItemsPage() {
  return (
    <RequireRole roles={["SUPER_ADMIN", "ADMIN", "HR"]}>
      <PayrollItemsPageContent />
    </RequireRole>
  );
}