"use client";

import { useCallback, useEffect, useState } from "react";

import { api, ApiError } from "@/lib/api-client";
import type { ListResponse, PaginationMeta } from "@/lib/types";

type Query = Record<string, string | number | boolean | undefined>;

/**
 * Fetches one page of a `{items, meta}` list endpoint, re-fetching when the
 * path, page, or filter query changes, and exposes a `reload()` escape
 * hatch for after a create/update/delete so the list reflects it without a
 * full page refresh.
 */
export function usePaginatedList<T>(path: string, query: Query = {}, pageSize = 20) {
  const [items, setItems] = useState<T[]>([]);
  const [meta, setMeta] = useState<PaginationMeta | null>(null);
  const [page, setPage] = useState(1);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [reloadToken, setReloadToken] = useState(0);

  // Query values are always primitives (strings/numbers/booleans/undefined)
  // in this app, so JSON.stringify is a safe, stable dependency key.
  const queryKey = JSON.stringify(query);

  useEffect(() => {
    // Resetting to page 1 when the filter query changes (not the page
    // itself) is the point of this effect — there's no way to do it
    // without a synchronous setState here.
    // eslint-disable-next-line react-hooks/set-state-in-effect
    setPage(1);
  }, [queryKey]);

  useEffect(() => {
    let cancelled = false;
    // Same as above: the skeleton/loading state must flip back on
    // immediately when path/page/filters change, not one render late.
    // eslint-disable-next-line react-hooks/set-state-in-effect
    setLoading(true);
    setError(null);

    api
      .get<ListResponse<T>>(path, { ...query, page, page_size: pageSize })
      .then((res) => {
        if (cancelled) return;
        setItems(res.items ?? []);
        setMeta(res.meta);
      })
      .catch((err: unknown) => {
        if (cancelled) return;
        setError(err instanceof ApiError ? err.message : "Gagal memuat data");
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });

    return () => {
      cancelled = true;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [path, page, pageSize, queryKey, reloadToken]);

  const reload = useCallback(() => setReloadToken((t) => t + 1), []);

  return { items, meta, page, setPage, loading, error, reload };
}
