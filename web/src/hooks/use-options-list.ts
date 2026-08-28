"use client";

import { useEffect, useState } from "react";

import { api } from "@/lib/api-client";
import type { ListResponse } from "@/lib/types";

/**
 * Loads every row of a list endpoint once, for populating a <Select> of
 * options (departments, positions, shifts, employees, ...) — these lists
 * are small enough in practice (tens, not thousands) that a single
 * page_size=100 request is simpler than wiring searchable/paginated
 * comboboxes for every relational field across the dashboard.
 */
export function useOptionsList<T>(path: string, enabled = true) {
  const [options, setOptions] = useState<T[]>([]);
  const [loading, setLoading] = useState(enabled);

  useEffect(() => {
    if (!enabled) return;
    let cancelled = false;
    // Deliberately resets loading synchronously when `path`/`enabled`
    // change (not just on mount): this hook can be re-pointed at a
    // different endpoint by the caller, and the consumer's skeleton state
    // needs to reflect that immediately, not one render late.
    // eslint-disable-next-line react-hooks/set-state-in-effect
    setLoading(true);
    api
      .get<ListResponse<T>>(path, { page: 1, page_size: 100 })
      .then((res) => {
        if (!cancelled) setOptions(res.items ?? []);
      })
      .catch(() => {
        if (!cancelled) setOptions([]);
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [path, enabled]);

  return { options, loading };
}
