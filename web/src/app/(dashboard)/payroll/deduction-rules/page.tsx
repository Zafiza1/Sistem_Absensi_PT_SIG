"use client";

import { useState, useEffect, useCallback } from "react";
import { Plus, Search, AlertTriangle, Trash2, Pencil, MoreHorizontal, Calculator } from "lucide-react";
import { toast } from "sonner";

import { api, ApiError } from "@/lib/api-client";
import { useAuth } from "@/lib/auth-context";
import { canWrite } from "@/lib/permissions";
import type { DeductionRule, DeductionRuleType, DeductionType } from "@/lib/types";

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
import { Switch } from "@/components/ui/switch";
import { ConfirmDeleteDialog } from "@/components/confirm-delete-dialog";
import { RequireRole } from "@/components/require-role";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";

const RULE_TYPE_LABELS: Record<DeductionRuleType, string> = {
  LATE: "Keterlambatan",
  ABSENT: "Ketidakhadiran",
  OTHER: "Lainnya",
};

const DEDUCTION_TYPE_LABELS: Record<DeductionType, string> = {
  FIXED: "Tetap",
  PERCENTAGE: "Persentase",
  HALF_DAY_SALARY: "Setengah Hari Gaji",
};

interface RuleForm {
  rule_type: DeductionRuleType;
  rule_name: string;
  description: string;
  late_min_minutes: number | null;
  late_max_minutes: number | null;
  deduction_amount: number;
  deduction_type: DeductionType;
  percentage_value: number | null;
  priority: number;
  is_active: boolean;
}

const EMPTY_RULE_FORM: RuleForm = {
  rule_type: "LATE",
  rule_name: "",
  description: "",
  late_min_minutes: null,
  late_max_minutes: null,
  deduction_amount: 0,
  deduction_type: "FIXED",
  percentage_value: null,
  priority: 0,
  is_active: true,
};

function formatCurrency(amount: number): string {
  return new Intl.NumberFormat("id-ID", {
    style: "currency",
    currency: "IDR",
    minimumFractionDigits: 0,
    maximumFractionDigits: 0,
  }).format(amount);
}

function DeductionRulesPageContent() {
  const { user } = useAuth();
  const writable = canWrite("payroll", user?.role);

  const [search, setSearch] = useState("");
  const [ruleTypeFilter, setRuleTypeFilter] = useState<DeductionRuleType | undefined>(undefined);
  const [activeFilter, setActiveFilter] = useState<boolean | undefined>(undefined);
  const [rules, setRules] = useState<DeductionRule[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const [dialogOpen, setDialogOpen] = useState(false);
  const [editing, setEditing] = useState<DeductionRule | null>(null);
  const [form, setForm] = useState<RuleForm>(EMPTY_RULE_FORM);
  const [saving, setSaving] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<DeductionRule | null>(null);

  // Load deduction rules
  const loadRules = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const params: Record<string, string | boolean> = {};
      if (ruleTypeFilter) params.rule_type = ruleTypeFilter;
      if (activeFilter !== undefined) params.is_active = activeFilter;

      const res = await api.get<{ items: DeductionRule[] }>("/payroll/deduction-rules", params);
      setRules(res.items);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Gagal memuat aturan potongan");
    } finally {
      setLoading(false);
    }
  }, [ruleTypeFilter, activeFilter]);

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect -- loading state must flip on immediately when filters change
    loadRules();
  }, [loadRules]);

  function openCreateDialog() {
    setEditing(null);
    setForm(EMPTY_RULE_FORM);
    setDialogOpen(true);
  }

  function openEditDialog(rule: DeductionRule) {
    setEditing(rule);
    setForm({
      rule_type: rule.rule_type,
      rule_name: rule.rule_name,
      description: rule.description ?? "",
      late_min_minutes: rule.late_min_minutes,
      late_max_minutes: rule.late_max_minutes,
      deduction_amount: rule.deduction_amount,
      deduction_type: rule.deduction_type,
      percentage_value: rule.percentage_value,
      priority: rule.priority,
      is_active: rule.is_active,
    });
    setDialogOpen(true);
  }

  async function handleSaveRule() {
    setSaving(true);
    try {
      if (editing) {
        await api.put(`/payroll/deduction-rules/${editing.id}`, form);
        toast.success("Aturan potongan berhasil diperbarui");
      } else {
        await api.post("/payroll/deduction-rules", form);
        toast.success("Aturan potongan berhasil dibuat");
      }
      setDialogOpen(false);
      loadRules();
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "Gagal menyimpan aturan potongan");
    } finally {
      setSaving(false);
    }
  }

  async function handleDeleteRule() {
    if (!deleteTarget) return;
    try {
      await api.delete(`/payroll/deduction-rules/${deleteTarget.id}`);
      toast.success("Aturan potongan berhasil dihapus");
      loadRules();
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "Gagal menghapus aturan potongan");
      throw err;
    }
  }

  const filteredRules = rules.filter(
    (rule) =>
      rule.rule_name.toLowerCase().includes(search.toLowerCase()) ||
      (rule.description && rule.description.toLowerCase().includes(search.toLowerCase())),
  );

  return (
    <div>
      <PageHeader
        title="Aturan Potongan"
        description="Kelola aturan potongan gaji untuk keterlambatan, ketidakhadiran, dan lainnya"
        action={
          writable && (
            <Button onClick={openCreateDialog}>
              <Plus className="size-4" />
              Tambah Aturan
            </Button>
          )
        }
      />

      <div className="mb-4 flex flex-wrap items-end gap-3 rounded-lg border bg-card p-4">
        <div className="relative max-w-sm flex-1">
          <Search className="pointer-events-none absolute top-1/2 left-2.5 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            placeholder="Cari aturan..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="pl-8"
          />
        </div>
        <div className="space-y-1.5">
          <Label className="text-xs text-muted-foreground">Tipe Aturan</Label>
          <Select
            items={{
              "": "Semua tipe",
              LATE: "Keterlambatan",
              ABSENT: "Ketidakhadiran",
              OTHER: "Lainnya",
            }}
            value={ruleTypeFilter ?? ""}
            onValueChange={(v) => setRuleTypeFilter((v as DeductionRuleType) || undefined)}
          >
            <SelectTrigger className="w-40">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="">Semua tipe</SelectItem>
              <SelectItem value="LATE">Keterlambatan</SelectItem>
              <SelectItem value="ABSENT">Ketidakhadiran</SelectItem>
              <SelectItem value="OTHER">Lainnya</SelectItem>
            </SelectContent>
          </Select>
        </div>
        <div className="space-y-1.5">
          <Label className="text-xs text-muted-foreground">Status</Label>
          <Select
            items={{ "": "Semua status", true: "Aktif", false: "Nonaktif" }}
            value={activeFilter === undefined ? "" : activeFilter.toString()}
            onValueChange={(v) => setActiveFilter(v === "" ? undefined : v === "true")}
          >
            <SelectTrigger className="w-32">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="">Semua status</SelectItem>
              <SelectItem value="true">Aktif</SelectItem>
              <SelectItem value="false">Nonaktif</SelectItem>
            </SelectContent>
          </Select>
        </div>
        <Button variant="outline" onClick={loadRules} className="ml-auto">
          Refresh
        </Button>
      </div>

      {/* Summary Cards */}
      <div className="mb-6 grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <Card className="flex-row items-center gap-3 p-4">
          <div className="flex size-10 shrink-0 items-center justify-center rounded-lg bg-primary/10 text-primary dark:bg-primary/20 dark:text-sky-400">
            <Calculator className="size-5" />
          </div>
          <div>
            <p className="text-xs text-muted-foreground">Total Aturan</p>
            <p className="text-xl font-semibold tracking-tight">{rules.length}</p>
          </div>
        </Card>
        <Card className="flex-row items-center gap-3 p-4">
          <div className="flex size-10 shrink-0 items-center justify-center rounded-lg bg-green-50 text-green-600 dark:bg-green-500/10 dark:text-green-400">
            <AlertTriangle className="size-5" />
          </div>
          <div>
            <p className="text-xs text-muted-foreground">Aturan Aktif</p>
            <p className="text-xl font-semibold tracking-tight">{rules.filter((r) => r.is_active).length}</p>
          </div>
        </Card>
        <Card className="flex-row items-center gap-3 p-4">
          <div className="flex size-10 shrink-0 items-center justify-center rounded-lg bg-amber-50 text-amber-600 dark:bg-amber-500/10 dark:text-amber-400">
            <AlertTriangle className="size-5" />
          </div>
          <div>
            <p className="text-xs text-muted-foreground">Keterlambatan</p>
            <p className="text-xl font-semibold tracking-tight">
              {rules.filter((r) => r.rule_type === "LATE").length}
            </p>
          </div>
        </Card>
        <Card className="flex-row items-center gap-3 p-4">
          <div className="flex size-10 shrink-0 items-center justify-center rounded-lg bg-rose-50 text-rose-600 dark:bg-rose-500/10 dark:text-rose-400">
            <AlertTriangle className="size-5" />
          </div>
          <div>
            <p className="text-xs text-muted-foreground">Ketidakhadiran</p>
            <p className="text-xl font-semibold tracking-tight">
              {rules.filter((r) => r.rule_type === "ABSENT").length}
            </p>
          </div>
        </Card>
      </div>

      <div className="rounded-lg border bg-card">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Nama Aturan</TableHead>
              <TableHead>Tipe</TableHead>
              <TableHead>Jenis Potongan</TableHead>
              <TableHead>Range Menit</TableHead>
              <TableHead>Jumlah Potongan</TableHead>
              <TableHead>Prioritas</TableHead>
              <TableHead>Status</TableHead>
              <TableHead>Berlaku Mulai</TableHead>
              {writable && <TableHead className="w-12" />}
            </TableRow>
          </TableHeader>
          <TableBody>
            {loading &&
              Array.from({ length: 5 }).map((_, i) => (
                <TableRow key={i}>
                  <TableCell colSpan={9}>
                    <Skeleton className="h-5 w-full" />
                  </TableCell>
                </TableRow>
              ))}
            {!loading && error && (
              <TableRow>
                <TableCell colSpan={9} className="py-8 text-center text-destructive">
                  {error}
                </TableCell>
              </TableRow>
            )}
            {!loading && !error && filteredRules.length === 0 && (
              <TableRow>
                <TableCell colSpan={9}>
                  <EmptyState
                    icon={Calculator}
                    title="Tidak ada aturan potongan ditemukan"
                    description="Mulai dengan membuat aturan potongan baru."
                  />
                </TableCell>
              </TableRow>
            )}
            {!loading &&
              !error &&
              filteredRules.map((rule) => (
                <TableRow key={rule.id}>
                  <TableCell>
                    <div>
                      <p className="font-medium">{rule.rule_name}</p>
                      {rule.description && (
                        <p className="text-xs text-muted-foreground">{rule.description}</p>
                      )}
                    </div>
                  </TableCell>
                  <TableCell>
                    <Badge variant="outline">{RULE_TYPE_LABELS[rule.rule_type]}</Badge>
                  </TableCell>
                  <TableCell>
                    <Badge variant="secondary">{DEDUCTION_TYPE_LABELS[rule.deduction_type]}</Badge>
                  </TableCell>
                  <TableCell>
                    {rule.late_min_minutes !== null || rule.late_max_minutes !== null ? (
                      <span className="text-sm">
                        {rule.late_min_minutes ?? 0} - {rule.late_max_minutes ?? "∞"} menit
                      </span>
                    ) : (
                      <span className="text-sm text-muted-foreground">-</span>
                    )}
                  </TableCell>
                  <TableCell>
                    {rule.deduction_type === "PERCENTAGE" ? (
                      <span className="font-medium">{rule.percentage_value}%</span>
                    ) : (
                      <span className="font-medium">{formatCurrency(rule.deduction_amount)}</span>
                    )}
                  </TableCell>
                  <TableCell>
                    <Badge variant="outline">{rule.priority}</Badge>
                  </TableCell>
                  <TableCell>
                    <Badge variant={rule.is_active ? "default" : "secondary"}>
                      {rule.is_active ? "Aktif" : "Nonaktif"}
                    </Badge>
                  </TableCell>
                  <TableCell>
                    <span className="text-sm text-muted-foreground">
                      {new Date(rule.effective_date).toLocaleDateString("id-ID")}
                    </span>
                  </TableCell>
                  {writable && (
                    <TableCell>
                      <DropdownMenu>
                        <DropdownMenuTrigger render={<Button variant="ghost" size="icon" className="size-8" />}>
                          <MoreHorizontal className="size-4" />
                        </DropdownMenuTrigger>
                        <DropdownMenuContent align="end">
                          <DropdownMenuItem onClick={() => openEditDialog(rule)}>
                            <Pencil className="mr-2 size-4" />
                            Edit
                          </DropdownMenuItem>
                          <DropdownMenuItem
                            variant="destructive"
                            onClick={() => setDeleteTarget(rule)}
                          >
                            <Trash2 className="mr-2 size-4" />
                            Hapus
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

      {/* Create Rule Dialog */}
      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <DialogContent className="max-h-[90vh] overflow-y-auto sm:max-w-lg">
          <DialogHeader>
            <DialogTitle>{editing ? "Edit Aturan Potongan" : "Tambah Aturan Potongan"}</DialogTitle>
          </DialogHeader>
          <div className="space-y-4">
            <div className="space-y-2">
              <Label>Tipe Aturan</Label>
              <Select
                items={{
                  LATE: "Keterlambatan",
                  ABSENT: "Ketidakhadiran",
                  OTHER: "Lainnya",
                }}
                value={form.rule_type}
                onValueChange={(v) => setForm((f) => ({ ...f, rule_type: v as DeductionRuleType }))}
              >
                <SelectTrigger className="w-full">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="LATE">Keterlambatan</SelectItem>
                  <SelectItem value="ABSENT">Ketidakhadiran</SelectItem>
                  <SelectItem value="OTHER">Lainnya</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-2">
              <Label htmlFor="rule_name">Nama Aturan</Label>
              <Input
                id="rule_name"
                value={form.rule_name}
                onChange={(e) => setForm((f) => ({ ...f, rule_name: e.target.value }))}
                placeholder="Contoh: Terlambat 1-10 Menit"
                required
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="description">Deskripsi (opsional)</Label>
              <Input
                id="description"
                value={form.description}
                onChange={(e) => setForm((f) => ({ ...f, description: e.target.value }))}
                placeholder="Penjelasan aturan ini"
              />
            </div>
            {form.rule_type === "LATE" && (
              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-2">
                  <Label htmlFor="late_min_minutes">Min Menit</Label>
                  <Input
                    id="late_min_minutes"
                    type="number"
                    value={form.late_min_minutes ?? ""}
                    onChange={(e) => setForm((f) => ({ ...f, late_min_minutes: e.target.value ? parseInt(e.target.value) : null }))}
                    placeholder="1"
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="late_max_minutes">Max Menit</Label>
                  <Input
                    id="late_max_minutes"
                    type="number"
                    value={form.late_max_minutes ?? ""}
                    onChange={(e) => setForm((f) => ({ ...f, late_max_minutes: e.target.value ? parseInt(e.target.value) : null }))}
                    placeholder="10"
                  />
                </div>
              </div>
            )}
            <div className="space-y-2">
              <Label>Jenis Potongan</Label>
              <Select
                items={{
                  FIXED: "Tetap",
                  PERCENTAGE: "Persentase",
                  HALF_DAY_SALARY: "Setengah Hari Gaji",
                }}
                value={form.deduction_type}
                onValueChange={(v) => setForm((f) => ({ ...f, deduction_type: v as DeductionType }))}
              >
                <SelectTrigger className="w-full">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="FIXED">Tetap</SelectItem>
                  <SelectItem value="PERCENTAGE">Persentase</SelectItem>
                  <SelectItem value="HALF_DAY_SALARY">Setengah Hari Gaji</SelectItem>
                </SelectContent>
              </Select>
            </div>
            {form.deduction_type === "PERCENTAGE" ? (
              <div className="space-y-2">
                <Label htmlFor="percentage_value">Persentase (%)</Label>
                <Input
                  id="percentage_value"
                  type="number"
                  value={form.percentage_value ?? ""}
                  onChange={(e) => setForm((f) => ({ ...f, percentage_value: e.target.value ? parseFloat(e.target.value) : null }))}
                  placeholder="10"
                />
              </div>
            ) : (
              <div className="space-y-2">
                <Label htmlFor="deduction_amount">Jumlah Potongan</Label>
                <Input
                  id="deduction_amount"
                  type="number"
                  value={form.deduction_amount}
                  onChange={(e) => setForm((f) => ({ ...f, deduction_amount: parseInt(e.target.value) }))}
                  placeholder="20000"
                />
              </div>
            )}
            <div className="space-y-2">
              <Label htmlFor="priority">Prioritas</Label>
              <Input
                id="priority"
                type="number"
                value={form.priority}
                onChange={(e) => setForm((f) => ({ ...f, priority: parseInt(e.target.value) }))}
                placeholder="0"
              />
              <p className="text-xs text-muted-foreground">
                Semakin tinggi prioritas, semakin diutamakan aturan ini
              </p>
            </div>
            {editing && (
              <div className="flex items-center justify-between rounded-md border p-3">
                <Label htmlFor="is_active" className="cursor-pointer">
                  Status aktif
                </Label>
                <Switch
                  id="is_active"
                  checked={form.is_active}
                  onCheckedChange={(checked) => setForm((f) => ({ ...f, is_active: checked }))}
                />
              </div>
            )}
          </div>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => setDialogOpen(false)}>
              Batal
            </Button>
            <Button type="button" onClick={handleSaveRule} disabled={saving}>
              {saving ? "Menyimpan..." : "Simpan"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <ConfirmDeleteDialog
        open={!!deleteTarget}
        onOpenChange={(open) => !open && setDeleteTarget(null)}
        title="Hapus aturan potongan?"
        description={`"${deleteTarget?.rule_name}" akan dihapus. Tindakan ini tidak dapat dibatalkan.`}
        onConfirm={handleDeleteRule}
      />
    </div>
  );
}

export default function DeductionRulesPage() {
  return (
    <RequireRole roles={["SUPER_ADMIN", "ADMIN", "HR"]}>
      <DeductionRulesPageContent />
    </RequireRole>
  );
}