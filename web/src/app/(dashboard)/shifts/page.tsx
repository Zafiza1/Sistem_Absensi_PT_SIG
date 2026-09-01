"use client";

import { useState, type FormEvent } from "react";
import { MoreHorizontal, Plus } from "lucide-react";
import { toast } from "sonner";

import { api, ApiError } from "@/lib/api-client";
import { useAuth } from "@/lib/auth-context";
import { canWrite } from "@/lib/permissions";
import { usePaginatedList } from "@/hooks/use-paginated-list";
import type { Shift } from "@/lib/types";

import { PageHeader } from "@/components/page-header";
import { DataTablePagination } from "@/components/data-table-pagination";
import { ConfirmDeleteDialog } from "@/components/confirm-delete-dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";

interface FormState {
  name: string;
  start_time: string;
  end_time: string;
  late_tolerance_minutes: string;
  working_duration_minutes: string;
  is_active: boolean;
}

const EMPTY_FORM: FormState = {
  name: "",
  start_time: "08:00",
  end_time: "17:00",
  late_tolerance_minutes: "15",
  working_duration_minutes: "480",
  is_active: true,
};

export default function ShiftsPage() {
  const { user } = useAuth();
  const writable = canWrite("shifts", user?.role);
  const { items, meta, setPage, loading, error, reload } = usePaginatedList<Shift>("/shifts");

  const [dialogOpen, setDialogOpen] = useState(false);
  const [editing, setEditing] = useState<Shift | null>(null);
  const [form, setForm] = useState<FormState>(EMPTY_FORM);
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const [saving, setSaving] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<Shift | null>(null);

  function openCreate() {
    setEditing(null);
    setForm(EMPTY_FORM);
    setFieldErrors({});
    setDialogOpen(true);
  }

  function openEdit(shift: Shift) {
    setEditing(shift);
    setForm({
      name: shift.name,
      start_time: shift.start_time,
      end_time: shift.end_time,
      late_tolerance_minutes: String(shift.late_tolerance_minutes),
      working_duration_minutes: String(shift.working_duration_minutes),
      is_active: shift.is_active,
    });
    setFieldErrors({});
    setDialogOpen(true);
  }

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setSaving(true);
    setFieldErrors({});
    const payload = {
      name: form.name,
      start_time: form.start_time,
      end_time: form.end_time,
      late_tolerance_minutes: Number(form.late_tolerance_minutes),
      working_duration_minutes: Number(form.working_duration_minutes),
      ...(editing ? { is_active: form.is_active } : {}),
    };
    try {
      if (editing) {
        await api.put(`/shifts/${editing.id}`, payload);
        toast.success("Shift berhasil diperbarui");
      } else {
        await api.post("/shifts", payload);
        toast.success("Shift berhasil dibuat");
      }
      setDialogOpen(false);
      reload();
    } catch (err) {
      if (err instanceof ApiError && err.fieldErrors) setFieldErrors(err.fieldErrors);
      else toast.error(err instanceof ApiError ? err.message : "Terjadi kesalahan");
    } finally {
      setSaving(false);
    }
  }

  async function handleDelete() {
    if (!deleteTarget) return;
    try {
      await api.delete(`/shifts/${deleteTarget.id}`);
      toast.success("Shift berhasil dihapus");
      reload();
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "Gagal menghapus data");
      throw err;
    }
  }

  return (
    <div>
      <PageHeader
        title="Shift"
        description="Definisi jam kerja & toleransi keterlambatan. Dipakai oleh Jam Kerja dan Jadwal Kerja untuk menentukan shift tiap hari."
        action={
          writable && (
            <Button onClick={openCreate}>
              <Plus className="size-4" />
              Tambah Shift
            </Button>
          )
        }
      />

      <div className="rounded-lg border bg-card">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Nama</TableHead>
              <TableHead>Jam Kerja</TableHead>
              <TableHead>Toleransi</TableHead>
              <TableHead>Durasi Kerja</TableHead>
              <TableHead>Status</TableHead>
              {writable && <TableHead className="w-12" />}
            </TableRow>
          </TableHeader>
          <TableBody>
            {loading &&
              Array.from({ length: 5 }).map((_, i) => (
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
            {!loading && !error && items.length === 0 && (
              <TableRow>
                <TableCell colSpan={6} className="py-8 text-center text-muted-foreground">
                  Belum ada data shift
                </TableCell>
              </TableRow>
            )}
            {!loading &&
              !error &&
              items.map((shift) => (
                <TableRow key={shift.id}>
                  <TableCell className="font-medium">
                    {shift.name}
                    {shift.is_overnight && (
                      <Badge variant="outline" className="ml-2 text-xs">
                        Lintas hari
                      </Badge>
                    )}
                  </TableCell>
                  <TableCell>
                    {shift.start_time} - {shift.end_time}
                  </TableCell>
                  <TableCell>{shift.late_tolerance_minutes} menit</TableCell>
                  <TableCell>{shift.working_duration_minutes} menit</TableCell>
                  <TableCell>
                    <Badge variant={shift.is_active ? "default" : "secondary"}>
                      {shift.is_active ? "Aktif" : "Nonaktif"}
                    </Badge>
                  </TableCell>
                  {writable && (
                    <TableCell>
                      <DropdownMenu>
                        <DropdownMenuTrigger render={<Button variant="ghost" size="icon" className="size-8" />}>
                          <MoreHorizontal className="size-4" />
                        </DropdownMenuTrigger>
                        <DropdownMenuContent align="end">
                          <DropdownMenuItem onClick={() => openEdit(shift)}>Edit</DropdownMenuItem>
                          <DropdownMenuItem variant="destructive" onClick={() => setDeleteTarget(shift)}>
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
        <DataTablePagination meta={meta} onPageChange={setPage} />
      </div>

      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{editing ? "Edit Shift" : "Tambah Shift"}</DialogTitle>
          </DialogHeader>
          <form onSubmit={handleSubmit} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="name">Nama Shift</Label>
              <Input id="name" value={form.name} onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))} required />
              {fieldErrors.name && <p className="text-sm text-destructive">{fieldErrors.name}</p>}
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label htmlFor="start_time">Jam Mulai</Label>
                <Input
                  id="start_time"
                  type="time"
                  value={form.start_time}
                  onChange={(e) => setForm((f) => ({ ...f, start_time: e.target.value }))}
                  required
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="end_time">Jam Selesai</Label>
                <Input
                  id="end_time"
                  type="time"
                  value={form.end_time}
                  onChange={(e) => setForm((f) => ({ ...f, end_time: e.target.value }))}
                  required
                />
              </div>
            </div>
            <p className="text-xs text-muted-foreground">
              Jika jam selesai lebih awal dari jam mulai, shift otomatis dianggap lintas hari (misal 22:00 → 06:00).
            </p>
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label htmlFor="late_tolerance_minutes">Toleransi Telat (menit)</Label>
                <Input
                  id="late_tolerance_minutes"
                  type="number"
                  min={0}
                  value={form.late_tolerance_minutes}
                  onChange={(e) => setForm((f) => ({ ...f, late_tolerance_minutes: e.target.value }))}
                  required
                />
                {fieldErrors.late_tolerance_minutes && (
                  <p className="text-sm text-destructive">{fieldErrors.late_tolerance_minutes}</p>
                )}
              </div>
              <div className="space-y-2">
                <Label htmlFor="working_duration_minutes">Durasi Kerja (menit)</Label>
                <Input
                  id="working_duration_minutes"
                  type="number"
                  min={1}
                  value={form.working_duration_minutes}
                  onChange={(e) => setForm((f) => ({ ...f, working_duration_minutes: e.target.value }))}
                  required
                />
                {fieldErrors.working_duration_minutes && (
                  <p className="text-sm text-destructive">{fieldErrors.working_duration_minutes}</p>
                )}
              </div>
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
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => setDialogOpen(false)}>
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
        open={!!deleteTarget}
        onOpenChange={(open) => !open && setDeleteTarget(null)}
        title="Hapus shift?"
        description={`"${deleteTarget?.name}" akan dihapus permanen. Tindakan ini tidak dapat dibatalkan.`}
        onConfirm={handleDelete}
      />
    </div>
  );
}
