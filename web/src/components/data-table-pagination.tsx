"use client";

import { ChevronLeft, ChevronRight } from "lucide-react";

import { Button } from "@/components/ui/button";
import type { PaginationMeta } from "@/lib/types";

export function DataTablePagination({
  meta,
  onPageChange,
}: {
  meta: PaginationMeta | null;
  onPageChange: (page: number) => void;
}) {
  if (!meta || meta.total_items === 0) return null;

  const from = (meta.page - 1) * meta.page_size + 1;
  const to = Math.min(meta.page * meta.page_size, meta.total_items);

  return (
    <div className="flex items-center justify-between border-t px-2 py-3">
      <p className="text-sm text-muted-foreground">
        Menampilkan {from}-{to} dari {meta.total_items} data
      </p>
      <div className="flex items-center gap-2">
        <Button
          variant="outline"
          size="sm"
          onClick={() => onPageChange(meta.page - 1)}
          disabled={meta.page <= 1}
        >
          <ChevronLeft className="size-4" />
          Sebelumnya
        </Button>
        <span className="text-sm text-muted-foreground">
          Halaman {meta.page} / {meta.total_pages}
        </span>
        <Button
          variant="outline"
          size="sm"
          onClick={() => onPageChange(meta.page + 1)}
          disabled={meta.page >= meta.total_pages}
        >
          Berikutnya
          <ChevronRight className="size-4" />
        </Button>
      </div>
    </div>
  );
}
