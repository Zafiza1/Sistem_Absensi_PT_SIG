"use client";

import { useState, type FormEvent } from "react";
import { MoreHorizontal, Plus } from "lucide-react";
import { toast } from "sonner";

import { api, ApiError } from "@/lib/api-client";
import { useAuth } from "@/lib/auth-context";
import { canWrite, type WritableResource } from "@/lib/permissions";
import { usePaginatedList } from "@/hooks/use-paginated-list";

import { PageHeader } from "@/components/page-header";
import { DataTablePagination } from "@/components/data-table-pagination";
import { ConfirmDeleteDialog } from "@/components/confirm-delete-dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
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

interface Entity {
  id: string;
  name: string;
  description: string;
  is_active: boolean;
}

interface FormState {
  name: string;
  description: string;
  is_active: boolean;
}

const EMPTY_FORM: FormState = { name: "", description: "", is_active: true };

/**
 * Shared list/create/edit/delete page body for the two master-data
 * resources shaped exactly like {name, description, is_active}:
 * departments and positions. Kept as one component rather than two nearly
 * identical page files.
 */
export function SimpleNameCrud({
  resource,
  endpoint,
  singularLabel,
  pluralLabel,
}: {
  resource: WritableResource;
  endpoint: string;
  singularLabel: string;
  pluralLabel: string;
}) {
  const { user } = useAuth();
  const writable = canWrite(resource, user?.role);
  const { items, meta, setPage, loading, error, reload } = usePaginatedList<Entity>(endpoint);

  const [dialogOpen, setDialogOpen] = useState(false);
  const [editing, setEditing] = useState<Entity | null>(null);
  const [form, setForm] = useState<FormState>(EMPTY_FORM);
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const [saving, setSaving] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<Entity | null>(null);

  function openCreate() {
    setEditing(null);
    setForm(EMPTY_FORM);
    setFieldErrors({});
    setDialogOpen(true);
  }

  function openEdit(entity: Entity) {
    setEditing(entity);
    setForm({ name: entity.name, description: entity.description, is_active: entity.is_active });
    setFieldErrors({});
    setDialogOpen(true);
  }

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setSaving(true);
    setFieldErrors({});
    try {
      if (editing) {
        await api.put(`${endpoint}/${editing.id}`, form);
        toast.success(`${singularLabel} berhasil diperbarui`);
      } else {
        await api.post(endpoint, form);
        toast.success(`${singularLabel} berhasil dibuat`);
      }
      setDialogOpen(false);
      reload();
    } catch (err) {
      if (err instanceof ApiError && err.fieldErrors) {
        setFieldErrors(err.fieldErrors);
      } else {
        toast.error(err instanceof ApiError ? err.message : "Terjadi kesalahan");
      }
    } finally {
      setSaving(false);
    }
  }

  async function handleDelete() {
    if (!deleteTarget) return;
    try {
      await api.delete(`${endpoint}/${deleteTarget.id}`);
      toast.success(`${singularLabel} berhasil dihapus`);
      reload();
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "Gagal menghapus data");
      throw err;
    }
  }

  return (
    <div>
      <PageHeader
        title={pluralLabel}
        description={`Kelola data ${pluralLabel.toLowerCase()}`}
        action={
          writable && (
            <Button onClick={openCreate}>
              <Plus className="size-4" />
              Tambah {singularLabel}
            </Button>
          )
        }
      />

      <div className="rounded-lg border bg-card">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Nama</TableHead>
              <TableHead>Deskripsi</TableHead>
              <TableHead>Status</TableHead>
              {writable && <TableHead className="w-12" />}
            </TableRow>
          </TableHeader>
          <TableBody>
            {loading &&
              Array.from({ length: 5 }).map((_, i) => (
                <TableRow key={i}>
                  <TableCell colSpan={writable ? 4 : 3}>
                    <Skeleton className="h-5 w-full" />
                  </TableCell>
                </TableRow>
              ))}
            {!loading && error && (
              <TableRow>
                <TableCell colSpan={writable ? 4 : 3} className="py-8 text-center text-destructive">
                  {error}
                </TableCell>
              </TableRow>
            )}
            {!loading && !error && items.length === 0 && (
              <TableRow>
                <TableCell colSpan={writable ? 4 : 3} className="py-8 text-center text-muted-foreground">
                  Belum ada data {pluralLabel.toLowerCase()}
                </TableCell>
              </TableRow>
            )}
            {!loading &&
              !error &&
              items.map((entity) => (
                <TableRow key={entity.id}>
                  <TableCell className="font-medium">{entity.name}</TableCell>
                  <TableCell className="max-w-md truncate text-muted-foreground">
                    {entity.description || "-"}
                  </TableCell>
                  <TableCell>
                    <Badge variant={entity.is_active ? "default" : "secondary"}>
                      {entity.is_active ? "Aktif" : "Nonaktif"}
                    </Badge>
                  </TableCell>
                  {writable && (
                    <TableCell>
                      <DropdownMenu>
                        <DropdownMenuTrigger render={<Button variant="ghost" size="icon" className="size-8" />}>
                          <MoreHorizontal className="size-4" />
                        </DropdownMenuTrigger>
                        <DropdownMenuContent align="end">
                          <DropdownMenuItem onClick={() => openEdit(entity)}>Edit</DropdownMenuItem>
                          <DropdownMenuItem variant="destructive" onClick={() => setDeleteTarget(entity)}>
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
            <DialogTitle>{editing ? `Edit ${singularLabel}` : `Tambah ${singularLabel}`}</DialogTitle>
          </DialogHeader>
          <form onSubmit={handleSubmit} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="name">Nama</Label>
              <Input
                id="name"
                value={form.name}
                onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))}
                required
              />
              {fieldErrors.name && <p className="text-sm text-destructive">{fieldErrors.name}</p>}
            </div>
            <div className="space-y-2">
              <Label htmlFor="description">Deskripsi</Label>
              <Textarea
                id="description"
                value={form.description}
                onChange={(e) => setForm((f) => ({ ...f, description: e.target.value }))}
                rows={3}
              />
              {fieldErrors.description && (
                <p className="text-sm text-destructive">{fieldErrors.description}</p>
              )}
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
        title={`Hapus ${singularLabel}?`}
        description={`"${deleteTarget?.name}" akan dihapus permanen. Tindakan ini tidak dapat dibatalkan.`}
        onConfirm={handleDelete}
      />
    </div>
  );
}
