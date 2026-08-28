"use client";

import { useState, type FormEvent } from "react";
import { MoreHorizontal, Plus } from "lucide-react";
import { toast } from "sonner";

import { api, ApiError } from "@/lib/api-client";
import { useAuth } from "@/lib/auth-context";
import { canWrite } from "@/lib/permissions";
import { usePaginatedList } from "@/hooks/use-paginated-list";
import { useOptionsList } from "@/hooks/use-options-list";
import type { Employee, Shift, WorkSchedule } from "@/lib/types";

import { PageHeader } from "@/components/page-header";
import { DataTablePagination } from "@/components/data-table-pagination";
import { ConfirmDeleteDialog } from "@/components/confirm-delete-dialog";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
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

const DAY_LABELS = ["Senin", "Selasa", "Rabu", "Kamis", "Jumat", "Sabtu", "Minggu"];

interface FormState {
  employee_id: string;
  shift_id: string;
  day_of_week: string;
}

const EMPTY_FORM: FormState = { employee_id: "", shift_id: "", day_of_week: "1" };

export default function SchedulesPage() {
  const { user } = useAuth();
  const writable = canWrite("schedules", user?.role);
  const { items, meta, setPage, loading, error, reload } = usePaginatedList<WorkSchedule>("/schedules");

  const { options: employees } = useOptionsList<Employee>("/employees");
  const { options: shifts } = useOptionsList<Shift>("/shifts");

  const [dialogOpen, setDialogOpen] = useState(false);
  const [editing, setEditing] = useState<WorkSchedule | null>(null);
  const [form, setForm] = useState<FormState>(EMPTY_FORM);
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const [saving, setSaving] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<WorkSchedule | null>(null);

  function openCreate() {
    setEditing(null);
    setForm(EMPTY_FORM);
    setFieldErrors({});
    setDialogOpen(true);
  }

  function openEdit(schedule: WorkSchedule) {
    setEditing(schedule);
    setForm({
      employee_id: schedule.employee_id,
      shift_id: schedule.shift_id,
      day_of_week: String(schedule.day_of_week),
    });
    setFieldErrors({});
    setDialogOpen(true);
  }

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setSaving(true);
    setFieldErrors({});
    try {
      if (editing) {
        await api.put(`/schedules/${editing.id}`, {
          shift_id: form.shift_id,
          day_of_week: Number(form.day_of_week),
        });
        toast.success("Jadwal berhasil diperbarui");
      } else {
        await api.post("/schedules", {
          employee_id: form.employee_id,
          shift_id: form.shift_id,
          day_of_week: Number(form.day_of_week),
        });
        toast.success("Jadwal berhasil dibuat");
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
      await api.delete(`/schedules/${deleteTarget.id}`);
      toast.success("Jadwal berhasil dihapus");
      reload();
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "Gagal menghapus data");
      throw err;
    }
  }

  return (
    <div>
      <PageHeader
        title="Jadwal Kerja"
        description="Atur shift khusus per hari untuk karyawan tertentu (mengganti shift default mereka)"
        action={
          writable && (
            <Button onClick={openCreate}>
              <Plus className="size-4" />
              Tambah Jadwal
            </Button>
          )
        }
      />

      <div className="rounded-lg border bg-card">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Karyawan</TableHead>
              <TableHead>Hari</TableHead>
              <TableHead>Shift</TableHead>
              {writable && <TableHead className="w-12" />}
            </TableRow>
          </TableHeader>
          <TableBody>
            {loading &&
              Array.from({ length: 5 }).map((_, i) => (
                <TableRow key={i}>
                  <TableCell colSpan={4}>
                    <Skeleton className="h-5 w-full" />
                  </TableCell>
                </TableRow>
              ))}
            {!loading && error && (
              <TableRow>
                <TableCell colSpan={4} className="py-8 text-center text-destructive">
                  {error}
                </TableCell>
              </TableRow>
            )}
            {!loading && !error && items.length === 0 && (
              <TableRow>
                <TableCell colSpan={4} className="py-8 text-center text-muted-foreground">
                  Belum ada jadwal khusus
                </TableCell>
              </TableRow>
            )}
            {!loading &&
              !error &&
              items.map((schedule) => (
                <TableRow key={schedule.id}>
                  <TableCell className="font-medium">{schedule.employee_name}</TableCell>
                  <TableCell>{DAY_LABELS[schedule.day_of_week - 1]}</TableCell>
                  <TableCell className="text-muted-foreground">{schedule.shift_name}</TableCell>
                  {writable && (
                    <TableCell>
                      <DropdownMenu>
                        <DropdownMenuTrigger render={<Button variant="ghost" size="icon" className="size-8" />}>
                          <MoreHorizontal className="size-4" />
                        </DropdownMenuTrigger>
                        <DropdownMenuContent align="end">
                          <DropdownMenuItem onClick={() => openEdit(schedule)}>Edit</DropdownMenuItem>
                          <DropdownMenuItem variant="destructive" onClick={() => setDeleteTarget(schedule)}>
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
            <DialogTitle>{editing ? "Edit Jadwal" : "Tambah Jadwal"}</DialogTitle>
          </DialogHeader>
          <form onSubmit={handleSubmit} className="space-y-4">
            <div className="space-y-2">
              <Label>Karyawan</Label>
              <Select
                items={Object.fromEntries(employees.map((e) => [e.id, `${e.name} (${e.employee_number})`]))}
                value={form.employee_id}
                onValueChange={(v) => setForm((f) => ({ ...f, employee_id: v ?? "" }))}
                disabled={!!editing}
              >
                <SelectTrigger className="w-full">
                  <SelectValue placeholder="Pilih karyawan" />
                </SelectTrigger>
                <SelectContent>
                  {employees.map((e) => (
                    <SelectItem key={e.id} value={e.id}>
                      {e.name} ({e.employee_number})
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              {fieldErrors.employee_id && <p className="text-sm text-destructive">{fieldErrors.employee_id}</p>}
            </div>
            <div className="space-y-2">
              <Label>Hari</Label>
              <Select
                items={Object.fromEntries(DAY_LABELS.map((label, i) => [String(i + 1), label]))}
                value={form.day_of_week}
                onValueChange={(v) => setForm((f) => ({ ...f, day_of_week: v ?? "1" }))}
              >
                <SelectTrigger className="w-full">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {DAY_LABELS.map((label, i) => (
                    <SelectItem key={label} value={String(i + 1)}>
                      {label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-2">
              <Label>Shift</Label>
              <Select
                items={Object.fromEntries(shifts.map((s) => [s.id, `${s.name} (${s.start_time}-${s.end_time})`]))}
                value={form.shift_id}
                onValueChange={(v) => setForm((f) => ({ ...f, shift_id: v ?? "" }))}
              >
                <SelectTrigger className="w-full">
                  <SelectValue placeholder="Pilih shift" />
                </SelectTrigger>
                <SelectContent>
                  {shifts.map((s) => (
                    <SelectItem key={s.id} value={s.id}>
                      {s.name} ({s.start_time}-{s.end_time})
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              {fieldErrors.shift_id && <p className="text-sm text-destructive">{fieldErrors.shift_id}</p>}
            </div>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => setDialogOpen(false)}>
                Batal
              </Button>
              <Button type="submit" disabled={saving || !form.employee_id || !form.shift_id}>
                {saving ? "Menyimpan..." : "Simpan"}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      <ConfirmDeleteDialog
        open={!!deleteTarget}
        onOpenChange={(open) => !open && setDeleteTarget(null)}
        title="Hapus jadwal?"
        description={`Jadwal ${deleteTarget ? DAY_LABELS[deleteTarget.day_of_week - 1] : ""} untuk "${deleteTarget?.employee_name}" akan dihapus.`}
        onConfirm={handleDelete}
      />
    </div>
  );
}
