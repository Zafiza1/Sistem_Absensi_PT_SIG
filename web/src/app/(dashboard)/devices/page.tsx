"use client";

import { useState, type FormEvent } from "react";
import { MoreHorizontal, Plus, Circle } from "lucide-react";
import { toast } from "sonner";

import { api, ApiError } from "@/lib/api-client";
import { useAuth } from "@/lib/auth-context";
import { canWrite } from "@/lib/permissions";
import { usePaginatedList } from "@/hooks/use-paginated-list";
import type { Device } from "@/lib/types";
import { cn } from "@/lib/utils";

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

interface FormState {
  device_name: string;
  device_code: string;
  location: string;
  status: "ACTIVE" | "INACTIVE";
}

const EMPTY_FORM: FormState = { device_name: "", device_code: "", location: "", status: "ACTIVE" };

function formatDate(value: string | null): string {
  if (!value) return "-";
  return new Date(value).toLocaleString("id-ID", { dateStyle: "medium", timeStyle: "short" });
}

export default function DevicesPage() {
  const { user } = useAuth();
  const writable = canWrite("devices", user?.role);
  const { items, meta, setPage, loading, error, reload } = usePaginatedList<Device>("/devices");

  const [dialogOpen, setDialogOpen] = useState(false);
  const [editing, setEditing] = useState<Device | null>(null);
  const [form, setForm] = useState<FormState>(EMPTY_FORM);
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const [saving, setSaving] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<Device | null>(null);

  function openCreate() {
    setEditing(null);
    setForm(EMPTY_FORM);
    setFieldErrors({});
    setDialogOpen(true);
  }

  function openEdit(device: Device) {
    setEditing(device);
    setForm({
      device_name: device.device_name,
      device_code: device.device_code,
      location: device.location,
      status: device.status,
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
        await api.put(`/devices/${editing.id}`, form);
        toast.success("Perangkat berhasil diperbarui");
      } else {
        await api.post("/devices/register", {
          device_name: form.device_name,
          device_code: form.device_code,
          location: form.location,
        });
        toast.success("Perangkat berhasil didaftarkan");
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
      await api.delete(`/devices/${deleteTarget.id}`);
      toast.success("Perangkat berhasil dihapus");
      reload();
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "Gagal menghapus data");
      throw err;
    }
  }

  return (
    <div>
      <PageHeader
        title="Perangkat"
        description="Kelola tablet absensi yang terdaftar"
        action={
          writable && (
            <Button onClick={openCreate}>
              <Plus className="size-4" />
              Daftarkan Perangkat
            </Button>
          )
        }
      />

      <div className="rounded-lg border bg-card">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Nama Perangkat</TableHead>
              <TableHead>Kode</TableHead>
              <TableHead>Lokasi</TableHead>
              <TableHead>Online</TableHead>
              <TableHead>Terakhir Aktif</TableHead>
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
                  Belum ada perangkat terdaftar
                </TableCell>
              </TableRow>
            )}
            {!loading &&
              !error &&
              items.map((device) => (
                <TableRow key={device.id}>
                  <TableCell className="font-medium">{device.device_name}</TableCell>
                  <TableCell className="font-mono text-sm">{device.device_code}</TableCell>
                  <TableCell className="text-muted-foreground">{device.location || "-"}</TableCell>
                  <TableCell>
                    <span className="flex items-center gap-1.5 text-sm">
                      <Circle
                        className={cn("size-2 fill-current", device.is_online ? "text-green-500" : "text-muted-foreground")}
                      />
                      {device.is_online ? "Online" : "Offline"}
                    </span>
                  </TableCell>
                  <TableCell className="text-sm text-muted-foreground">{formatDate(device.last_seen_at)}</TableCell>
                  <TableCell>
                    <Badge variant={device.status === "ACTIVE" ? "default" : "secondary"}>
                      {device.status === "ACTIVE" ? "Aktif" : "Nonaktif"}
                    </Badge>
                  </TableCell>
                  {writable && (
                    <TableCell>
                      <DropdownMenu>
                        <DropdownMenuTrigger render={<Button variant="ghost" size="icon" className="size-8" />}>
                          <MoreHorizontal className="size-4" />
                        </DropdownMenuTrigger>
                        <DropdownMenuContent align="end">
                          <DropdownMenuItem onClick={() => openEdit(device)}>Edit</DropdownMenuItem>
                          <DropdownMenuItem variant="destructive" onClick={() => setDeleteTarget(device)}>
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
            <DialogTitle>{editing ? "Edit Perangkat" : "Daftarkan Perangkat"}</DialogTitle>
          </DialogHeader>
          <form onSubmit={handleSubmit} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="device_name">Nama Perangkat</Label>
              <Input
                id="device_name"
                value={form.device_name}
                onChange={(e) => setForm((f) => ({ ...f, device_name: e.target.value }))}
                required
              />
              {fieldErrors.device_name && <p className="text-sm text-destructive">{fieldErrors.device_name}</p>}
            </div>
            <div className="space-y-2">
              <Label htmlFor="device_code">Kode Perangkat</Label>
              <Input
                id="device_code"
                placeholder="TAB-001"
                value={form.device_code}
                onChange={(e) => setForm((f) => ({ ...f, device_code: e.target.value }))}
                required
              />
              <p className="text-xs text-muted-foreground">
                Kode ini yang dimasukkan di aplikasi tablet saat pendaftaran perangkat.
              </p>
              {fieldErrors.device_code && <p className="text-sm text-destructive">{fieldErrors.device_code}</p>}
            </div>
            <div className="space-y-2">
              <Label htmlFor="location">Lokasi</Label>
              <Input
                id="location"
                value={form.location}
                onChange={(e) => setForm((f) => ({ ...f, location: e.target.value }))}
              />
            </div>
            {editing && (
              <div className="space-y-2">
                <Label htmlFor="status">Status</Label>
                <Select
                  items={{ ACTIVE: "Aktif", INACTIVE: "Nonaktif" }}
                  value={form.status}
                  onValueChange={(v) => setForm((f) => ({ ...f, status: v as FormState["status"] }))}
                >
                  <SelectTrigger id="status" className="w-full">
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
        title="Hapus perangkat?"
        description={`"${deleteTarget?.device_name}" akan dihapus permanen. Tablet ini tidak akan bisa lagi mencatat absensi.`}
        onConfirm={handleDelete}
      />
    </div>
  );
}
