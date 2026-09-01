"use client";

import { useEffect, useState } from "react";
import { toast } from "sonner";

import { api, ApiError } from "@/lib/api-client";
import { useAuth } from "@/lib/auth-context";
import { canWrite } from "@/lib/permissions";
import { useOptionsList } from "@/hooks/use-options-list";
import type { CompanyScheduleDay, Shift } from "@/lib/types";

import { PageHeader } from "@/components/page-header";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";

const DAYS = [
  { value: 1, label: "Senin" },
  { value: 2, label: "Selasa" },
  { value: 3, label: "Rabu" },
  { value: 4, label: "Kamis" },
  { value: 5, label: "Jumat" },
  { value: 6, label: "Sabtu" },
  { value: 7, label: "Minggu" },
];

const OFF = "OFF";

export default function CompanySchedulePage() {
  const { user } = useAuth();
  const writable = canWrite("company-schedule", user?.role);
  const { options: shifts, loading: shiftsLoading } = useOptionsList<Shift>("/shifts");

  // day_of_week (1..7) -> selected value: a shift id, or OFF for a
  // non-working day. Every day defaults to OFF until the API says otherwise.
  const [rows, setRows] = useState<Record<number, string>>(() =>
    Object.fromEntries(DAYS.map((d) => [d.value, OFF])),
  );
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [reloadToken, setReloadToken] = useState(0);

  useEffect(() => {
    let cancelled = false;
    // eslint-disable-next-line react-hooks/set-state-in-effect
    setLoading(true);

    api
      .get<{ days: CompanyScheduleDay[] }>("/company-schedule")
      .then((data) => {
        if (cancelled) return;
        setRows((prev) => {
          const next = { ...prev };
          for (const day of data.days ?? []) {
            next[day.day_of_week] = day.shift_id ?? OFF;
          }
          return next;
        });
      })
      .catch((err: unknown) => {
        if (!cancelled) toast.error(err instanceof ApiError ? err.message : "Gagal memuat jadwal kerja");
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });

    return () => {
      cancelled = true;
    };
  }, [reloadToken]);

  async function handleSave() {
    setSaving(true);
    try {
      const payload = {
        days: DAYS.map((d) => ({
          day_of_week: d.value,
          shift_id: rows[d.value] === OFF ? null : rows[d.value],
        })),
      };
      await api.put("/company-schedule", payload);
      toast.success("Jadwal kerja perusahaan berhasil disimpan");
      setReloadToken((t) => t + 1);
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "Terjadi kesalahan");
    } finally {
      setSaving(false);
    }
  }

  const busy = loading || shiftsLoading;

  return (
    <div>
      <PageHeader
        title="Jam Kerja"
        description="Jadwal kerja default seluruh karyawan per hari. Karyawan dengan jam berbeda diatur di menu Jadwal Kerja."
        action={
          writable && (
            <Button onClick={handleSave} disabled={busy || saving}>
              {saving ? "Menyimpan..." : "Simpan"}
            </Button>
          )
        }
      />

      <div className="max-w-xl rounded-lg border bg-card divide-y">
        {busy &&
          DAYS.map((d) => (
            <div key={d.value} className="flex items-center justify-between gap-4 p-4">
              <Skeleton className="h-5 w-20" />
              <Skeleton className="h-8 w-56" />
            </div>
          ))}

        {!busy &&
          DAYS.map((d) => {
            const items: Record<string, string> = {
              [OFF]: "Libur",
              ...Object.fromEntries(shifts.map((s) => [s.id, `${s.name} (${s.start_time}-${s.end_time})`])),
            };
            return (
              <div key={d.value} className="flex items-center justify-between gap-4 p-4">
                <span className="text-sm font-medium">{d.label}</span>
                <Select
                  items={items}
                  value={rows[d.value]}
                  onValueChange={(v) => setRows((r) => ({ ...r, [d.value]: v ?? OFF }))}
                  disabled={!writable}
                >
                  <SelectTrigger className="w-56">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value={OFF}>Libur</SelectItem>
                    {shifts.map((s) => (
                      <SelectItem key={s.id} value={s.id}>
                        {s.name} ({s.start_time}-{s.end_time})
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
            );
          })}
      </div>

      <p className="mt-4 max-w-xl text-xs text-muted-foreground">
        Urutan penentuan shift saat absensi: jadwal khusus karyawan (menu Jadwal Kerja) &rarr; jadwal
        kerja di halaman ini &rarr; shift default karyawan. Hari yang disetel &ldquo;Libur&rdquo; akan
        menolak absensi kecuali karyawan tersebut punya jadwal khusus hari itu.
      </p>
    </div>
  );
}
