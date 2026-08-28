"use client";

import { useState, type FormEvent } from "react";
import { MoreHorizontal, Plus, Copy, Check } from "lucide-react";
import { toast } from "sonner";

import { api, ApiError } from "@/lib/api-client";
import { usePaginatedList } from "@/hooks/use-paginated-list";
import { ROLES, ROLE_LABELS, type DashboardUser, type Role } from "@/lib/types";

import { RequireRole } from "@/components/require-role";
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
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle, DialogDescription } from "@/components/ui/dialog";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";

interface FormState {
  name: string;
  email: string;
  role: Role;
  password: string;
  is_active: boolean;
}

const EMPTY_FORM: FormState = { name: "", email: "", role: "HR", password: "", is_active: true };

function GeneratedPasswordDialog({
  password,
  onOpenChange,
}: {
  password: string | null;
  onOpenChange: (open: boolean) => void;
}) {
  const [copied, setCopied] = useState(false);

  async function handleCopy() {
    if (!password) return;
    try {
      await navigator.clipboard.writeText(password);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      toast.error("Gagal menyalin ke clipboard");
    }
  }

  return (
    <AlertDialog open={!!password} onOpenChange={onOpenChange}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>Password Sementara</AlertDialogTitle>
          <AlertDialogDescription>
            Simpan dan bagikan password ini ke pengguna secara aman. Password ini tidak akan ditampilkan lagi setelah
            dialog ini ditutup.
          </AlertDialogDescription>
        </AlertDialogHeader>
        <div className="flex items-center gap-2">
          <Input readOnly value={password ?? ""} className="font-mono" />
          <Button type="button" variant="outline" size="icon" onClick={handleCopy}>
            {copied ? <Check className="size-4" /> : <Copy className="size-4" />}
          </Button>
        </div>
        <AlertDialogFooter>
          <AlertDialogAction onClick={() => onOpenChange(false)}>Selesai</AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}

function UsersPageContent() {
  const { items, meta, setPage, loading, error, reload } = usePaginatedList<DashboardUser>("/users");

  const [dialogOpen, setDialogOpen] = useState(false);
  const [editing, setEditing] = useState<DashboardUser | null>(null);
  const [form, setForm] = useState<FormState>(EMPTY_FORM);
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const [saving, setSaving] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<DashboardUser | null>(null);
  const [generatedPassword, setGeneratedPassword] = useState<string | null>(null);

  function openCreate() {
    setEditing(null);
    setForm(EMPTY_FORM);
    setFieldErrors({});
    setDialogOpen(true);
  }

  function openEdit(u: DashboardUser) {
    setEditing(u);
    setForm({ name: u.name, email: u.email, role: u.role, password: "", is_active: u.is_active });
    setFieldErrors({});
    setDialogOpen(true);
  }

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setSaving(true);
    setFieldErrors({});
    try {
      if (editing) {
        await api.put(`/users/${editing.id}`, {
          name: form.name,
          email: form.email,
          role: form.role,
          is_active: form.is_active,
        });
        toast.success("Akun berhasil diperbarui");
        setDialogOpen(false);
      } else {
        const created = await api.post<DashboardUser>("/users", {
          name: form.name,
          email: form.email,
          role: form.role,
          password: form.password || undefined,
        });
        toast.success("Akun berhasil dibuat");
        setDialogOpen(false);
        if (created.generated_password) setGeneratedPassword(created.generated_password);
      }
      reload();
    } catch (err) {
      if (err instanceof ApiError && err.fieldErrors) setFieldErrors(err.fieldErrors);
      else toast.error(err instanceof ApiError ? err.message : "Terjadi kesalahan");
    } finally {
      setSaving(false);
    }
  }

  async function handleResetPassword(u: DashboardUser) {
    try {
      const result = await api.post<{ generated_password: string }>(`/users/${u.id}/reset-password`);
      setGeneratedPassword(result.generated_password);
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "Gagal mereset password");
    }
  }

  async function handleDelete() {
    if (!deleteTarget) return;
    try {
      await api.delete(`/users/${deleteTarget.id}`);
      toast.success("Akun berhasil dinonaktifkan");
      reload();
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "Gagal menghapus akun");
      throw err;
    }
  }

  return (
    <div>
      <PageHeader
        title="Kelola Akun"
        description="Akun dashboard untuk Admin, HR, dan Management"
        action={
          <Button onClick={openCreate}>
            <Plus className="size-4" />
            Tambah Akun
          </Button>
        }
      />

      <div className="rounded-lg border bg-card">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Nama</TableHead>
              <TableHead>Email</TableHead>
              <TableHead>Peran</TableHead>
              <TableHead>Status</TableHead>
              <TableHead className="w-12" />
            </TableRow>
          </TableHeader>
          <TableBody>
            {loading &&
              Array.from({ length: 5 }).map((_, i) => (
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
            {!loading &&
              !error &&
              items.map((u) => (
                <TableRow key={u.id}>
                  <TableCell className="font-medium">{u.name}</TableCell>
                  <TableCell className="text-muted-foreground">{u.email}</TableCell>
                  <TableCell>
                    <Badge variant="outline">{ROLE_LABELS[u.role]}</Badge>
                  </TableCell>
                  <TableCell>
                    <Badge variant={u.is_active ? "default" : "secondary"}>{u.is_active ? "Aktif" : "Nonaktif"}</Badge>
                  </TableCell>
                  <TableCell>
                    <DropdownMenu>
                      <DropdownMenuTrigger render={<Button variant="ghost" size="icon" className="size-8" />}>
                        <MoreHorizontal className="size-4" />
                      </DropdownMenuTrigger>
                      <DropdownMenuContent align="end">
                        <DropdownMenuItem onClick={() => openEdit(u)}>Edit</DropdownMenuItem>
                        <DropdownMenuItem onClick={() => handleResetPassword(u)}>Reset Password</DropdownMenuItem>
                        <DropdownMenuItem variant="destructive" onClick={() => setDeleteTarget(u)}>
                          Nonaktifkan
                        </DropdownMenuItem>
                      </DropdownMenuContent>
                    </DropdownMenu>
                  </TableCell>
                </TableRow>
              ))}
          </TableBody>
        </Table>
        <DataTablePagination meta={meta} onPageChange={setPage} />
      </div>

      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{editing ? "Edit Akun" : "Tambah Akun"}</DialogTitle>
            {!editing && (
              <DialogDescription>
                Kosongkan password untuk membuat password acak yang akan ditampilkan setelah akun dibuat.
              </DialogDescription>
            )}
          </DialogHeader>
          <form onSubmit={handleSubmit} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="name">Nama</Label>
              <Input id="name" value={form.name} onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))} required />
              {fieldErrors.name && <p className="text-sm text-destructive">{fieldErrors.name}</p>}
            </div>
            <div className="space-y-2">
              <Label htmlFor="email">Email</Label>
              <Input
                id="email"
                type="email"
                value={form.email}
                onChange={(e) => setForm((f) => ({ ...f, email: e.target.value }))}
                required
              />
              {fieldErrors.email && <p className="text-sm text-destructive">{fieldErrors.email}</p>}
            </div>
            <div className="space-y-2">
              <Label>Peran</Label>
              <Select
                items={ROLE_LABELS}
                value={form.role}
                onValueChange={(v) => setForm((f) => ({ ...f, role: (v ?? f.role) as Role }))}
              >
                <SelectTrigger className="w-full">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {ROLES.map((r) => (
                    <SelectItem key={r} value={r}>
                      {ROLE_LABELS[r]}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            {!editing && (
              <div className="space-y-2">
                <Label htmlFor="password">Password (opsional)</Label>
                <Input
                  id="password"
                  type="password"
                  value={form.password}
                  onChange={(e) => setForm((f) => ({ ...f, password: e.target.value }))}
                  placeholder="Kosongkan untuk password acak"
                />
                {fieldErrors.password && <p className="text-sm text-destructive">{fieldErrors.password}</p>}
              </div>
            )}
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

      <GeneratedPasswordDialog password={generatedPassword} onOpenChange={(open) => !open && setGeneratedPassword(null)} />

      <ConfirmDeleteDialog
        open={!!deleteTarget}
        onOpenChange={(open) => !open && setDeleteTarget(null)}
        title="Nonaktifkan akun?"
        description={`"${deleteTarget?.name}" tidak akan bisa lagi login ke dashboard.`}
        onConfirm={handleDelete}
      />
    </div>
  );
}

export default function UsersPage() {
  return (
    <RequireRole roles={["SUPER_ADMIN"]}>
      <UsersPageContent />
    </RequireRole>
  );
}
