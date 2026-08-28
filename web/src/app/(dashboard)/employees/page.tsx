"use client";

import { useState, type FormEvent } from "react";
import { MoreHorizontal, Plus, Search } from "lucide-react";
import { toast } from "sonner";

import { api, ApiError } from "@/lib/api-client";
import { useAuth } from "@/lib/auth-context";
import { canWrite } from "@/lib/permissions";
import { usePaginatedList } from "@/hooks/use-paginated-list";
import { useOptionsList } from "@/hooks/use-options-list";
import type { Department, Employee, Position, Shift } from "@/lib/types";

import { PageHeader } from "@/components/page-header";
import { DataTablePagination } from "@/components/data-table-pagination";
import { ConfirmDeleteDialog } from "@/components/confirm-delete-dialog";
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

const UNASSIGNED = "__unassigned__";

interface FormState {
  employee_number: string;
  name: string;
  email: string;
  phone: string;
  department_id: string;
  position_id: string;
  shift_id: string;
  status: "ACTIVE" | "INACTIVE";
}

const EMPTY_FORM: FormState = {
  employee_number: "",
  name: "",
  email: "",
  phone: "",
  department_id: UNASSIGNED,
  position_id: UNASSIGNED,
  shift_id: UNASSIGNED,
  status: "ACTIVE",
};

export default function EmployeesPage() {
  const { user } = useAuth();
  const writable = canWrite("employees", user?.role);

  const [search, setSearch] = useState("");
  const { items, meta, setPage, loading, error, reload } = usePaginatedList<Employee>("/employees", {
    search: search || undefined,
  });

  const { options: departments } = useOptionsList<Department>("/departments");
  const { options: positions } = useOptionsList<Position>("/positions");
  const { options: shifts } = useOptionsList<Shift>("/shifts");

  const [dialogOpen, setDialogOpen] = useState(false);
  const [editing, setEditing] = useState<Employee | null>(null);
  const [form, setForm] = useState<FormState>(EMPTY_FORM);
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const [saving, setSaving] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<Employee | null>(null);

  function openCreate() {
    setEditing(null);
    setForm(EMPTY_FORM);
    setFieldErrors({});
    setDialogOpen(true);
  }

  function openEdit(employee: Employee) {
    setEditing(employee);
    setForm({
      employee_number: employee.employee_number,
      name: employee.name,
      email: employee.email ?? "",
      phone: employee.phone ?? "",
      department_id: employee.department_id ?? UNASSIGNED,
      position_id: employee.position_id ?? UNASSIGNED,
      shift_id: employee.shift_id ?? UNASSIGNED,
      status: employee.status,
    });
    setFieldErrors({});
    setDialogOpen(true);
  }

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setSaving(true);
    setFieldErrors({});
    const payload = {
      employee_number: form.employee_number,
      name: form.name,
      email: form.email || null,
      phone: form.phone || null,
      department_id: form.department_id === UNASSIGNED ? null : form.department_id,
      position_id: form.position_id === UNASSIGNED ? null : form.position_id,
      shift_id: form.shift_id === UNASSIGNED ? null : form.shift_id,
      ...(editing ? { status: form.status } : {}),
    };
    try {
      if (editing) {
        await api.put(`/employees/${editing.id}`, payload);
        toast.success("Karyawan berhasil diperbarui");
      } else {
        await api.post("/employees", payload);
        toast.success("Karyawan berhasil ditambahkan");
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
      await api.delete(`/employees/${deleteTarget.id}`);
      toast.success("Karyawan berhasil dinonaktifkan");
      reload();
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "Gagal menghapus data");
      throw err;
    }
  }

  return (
    <div>
      <PageHeader
        title="Karyawan"
        description="Kelola data karyawan dan penempatan shift"
        action={
          writable && (
            <Button onClick={openCreate}>
              <Plus className="size-4" />
              Tambah Karyawan
            </Button>
          )
        }
      />

      <div className="mb-4 max-w-sm">
        <div className="relative">
          <Search className="pointer-events-none absolute top-1/2 left-2.5 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            placeholder="Cari nama atau NIK..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="pl-8"
          />
        </div>
      </div>

      <div className="rounded-lg border bg-card">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>NIK</TableHead>
              <TableHead>Nama</TableHead>
              <TableHead>Departemen</TableHead>
              <TableHead>Jabatan</TableHead>
              <TableHead>Shift</TableHead>
              <TableHead>Status</TableHead>
              {writable && <TableHead className="w-12" />}
            </TableRow>
          </TableHeader>
          <TableBody>
            {loading &&
              Array.from({ length: 5 }).map((_, i) => (
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
                  Tidak ada karyawan ditemukan
                </TableCell>
              </TableRow>
            )}
            {!loading &&
              !error &&
              items.map((employee) => (
                <TableRow key={employee.id}>
                  <TableCell className="font-mono text-sm">{employee.employee_number}</TableCell>
                  <TableCell className="font-medium">{employee.name}</TableCell>
                  <TableCell className="text-muted-foreground">{employee.department_name ?? "-"}</TableCell>
                  <TableCell className="text-muted-foreground">{employee.position_name ?? "-"}</TableCell>
                  <TableCell className="text-muted-foreground">{employee.shift_name ?? "-"}</TableCell>
                  <TableCell>
                    <Badge variant={employee.status === "ACTIVE" ? "default" : "secondary"}>
                      {employee.status === "ACTIVE" ? "Aktif" : "Nonaktif"}
                    </Badge>
                  </TableCell>
                  {writable && (
                    <TableCell>
                      <DropdownMenu>
                        <DropdownMenuTrigger render={<Button variant="ghost" size="icon" className="size-8" />}>
                          <MoreHorizontal className="size-4" />
                        </DropdownMenuTrigger>
                        <DropdownMenuContent align="end">
                          <DropdownMenuItem onClick={() => openEdit(employee)}>Edit</DropdownMenuItem>
                          <DropdownMenuItem variant="destructive" onClick={() => setDeleteTarget(employee)}>
                            Nonaktifkan
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
        <DialogContent className="max-h-[90vh] overflow-y-auto sm:max-w-lg">
          <DialogHeader>
            <DialogTitle>{editing ? "Edit Karyawan" : "Tambah Karyawan"}</DialogTitle>
          </DialogHeader>
          <form onSubmit={handleSubmit} className="space-y-4">
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label htmlFor="employee_number">NIK</Label>
                <Input
                  id="employee_number"
                  value={form.employee_number}
                  onChange={(e) => setForm((f) => ({ ...f, employee_number: e.target.value }))}
                  required
                />
                {fieldErrors.employee_number && (
                  <p className="text-sm text-destructive">{fieldErrors.employee_number}</p>
                )}
              </div>
              <div className="space-y-2">
                <Label htmlFor="name">Nama</Label>
                <Input id="name" value={form.name} onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))} required />
                {fieldErrors.name && <p className="text-sm text-destructive">{fieldErrors.name}</p>}
              </div>
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label htmlFor="email">Email</Label>
                <Input
                  id="email"
                  type="email"
                  value={form.email}
                  onChange={(e) => setForm((f) => ({ ...f, email: e.target.value }))}
                />
                {fieldErrors.email && <p className="text-sm text-destructive">{fieldErrors.email}</p>}
              </div>
              <div className="space-y-2">
                <Label htmlFor="phone">Telepon</Label>
                <Input id="phone" value={form.phone} onChange={(e) => setForm((f) => ({ ...f, phone: e.target.value }))} />
              </div>
            </div>
            <div className="space-y-2">
              <Label>Departemen</Label>
              <Select
                items={{ [UNASSIGNED]: "Belum ditentukan", ...Object.fromEntries(departments.map((d) => [d.id, d.name])) }}
                value={form.department_id}
                onValueChange={(v) => setForm((f) => ({ ...f, department_id: v ?? UNASSIGNED }))}
              >
                <SelectTrigger className="w-full">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value={UNASSIGNED}>Belum ditentukan</SelectItem>
                  {departments.map((d) => (
                    <SelectItem key={d.id} value={d.id}>
                      {d.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-2">
              <Label>Jabatan</Label>
              <Select
                items={{ [UNASSIGNED]: "Belum ditentukan", ...Object.fromEntries(positions.map((p) => [p.id, p.name])) }}
                value={form.position_id}
                onValueChange={(v) => setForm((f) => ({ ...f, position_id: v ?? UNASSIGNED }))}
              >
                <SelectTrigger className="w-full">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value={UNASSIGNED}>Belum ditentukan</SelectItem>
                  {positions.map((p) => (
                    <SelectItem key={p.id} value={p.id}>
                      {p.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-2">
              <Label>Shift Default</Label>
              <Select
                items={{
                  [UNASSIGNED]: "Belum ditentukan",
                  ...Object.fromEntries(shifts.map((s) => [s.id, `${s.name} (${s.start_time}-${s.end_time})`])),
                }}
                value={form.shift_id}
                onValueChange={(v) => setForm((f) => ({ ...f, shift_id: v ?? UNASSIGNED }))}
              >
                <SelectTrigger className="w-full">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value={UNASSIGNED}>Belum ditentukan</SelectItem>
                  {shifts.map((s) => (
                    <SelectItem key={s.id} value={s.id}>
                      {s.name} ({s.start_time}-{s.end_time})
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              <p className="text-xs text-muted-foreground">
                Dipakai jika karyawan tidak punya jadwal khusus untuk hari tersebut (lihat halaman Jadwal Kerja).
              </p>
            </div>
            {editing && (
              <div className="space-y-2">
                <Label>Status</Label>
                <Select
                  items={{ ACTIVE: "Aktif", INACTIVE: "Nonaktif" }}
                  value={form.status}
                  onValueChange={(v) => setForm((f) => ({ ...f, status: v as FormState["status"] }))}
                >
                  <SelectTrigger className="w-full">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="ACTIVE">Aktif</SelectItem>
                    <SelectItem value="INACTIVE">Nonaktif</SelectItem>
                  </SelectContent>
                </Select>
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
        title="Nonaktifkan karyawan?"
        description={`"${deleteTarget?.name}" akan dinonaktifkan. Riwayat absensinya tetap tersimpan.`}
        onConfirm={handleDelete}
      />
    </div>
  );
}
