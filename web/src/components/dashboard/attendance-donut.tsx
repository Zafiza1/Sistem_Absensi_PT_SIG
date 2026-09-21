"use client";

import { useState } from "react";

import { cn } from "@/lib/utils";
import { Skeleton } from "@/components/ui/skeleton";

export interface DonutSegment {
  label: string;
  value: number;
  colorClass: string; // text-* tailwind class, applied to the arc via currentColor
  dotClass: string; // bg-* tailwind class for the legend swatch
}

const SIZE = 176;
const STROKE = 26;
const RADIUS = (SIZE - STROKE) / 2;
const CIRCUMFERENCE = 2 * Math.PI * RADIUS;
const GAP = 3; // px of surface gap between adjacent segments

export function AttendanceDonut({
  segments,
  total,
  centerLabel = "Total Karyawan",
  loading,
}: {
  segments: DonutSegment[];
  total: number;
  centerLabel?: string;
  loading: boolean;
}) {
  const [hovered, setHovered] = useState<number | null>(null);

  if (loading) {
    return (
      <div className="flex items-center gap-6">
        <Skeleton className="size-44 shrink-0 rounded-full" />
        <div className="flex-1 space-y-2.5">
          {Array.from({ length: 3 }).map((_, i) => (
            <Skeleton key={i} className="h-4 w-full" />
          ))}
        </div>
      </div>
    );
  }

  const safeTotal = total || 1;
  const active = hovered !== null ? segments[hovered] : null;

  // Offsets computed as a pure fold (each step returns a new array, no
  // mutation of any existing binding) so nothing is reassigned during render.
  const offsets = segments.reduce<number[]>((acc, _seg, i) => {
    const prevOffset = i === 0 ? 0 : acc[i - 1] + (segments[i - 1].value / safeTotal) * CIRCUMFERENCE;
    return [...acc, prevOffset];
  }, []);

  return (
    <div className="flex flex-wrap items-center gap-6">
      <div className="relative shrink-0" role="img" aria-label={`${centerLabel}: ${total}, terdiri dari ${segments.map((s) => `${s.label} ${s.value}`).join(", ")}`}>
        <svg width={SIZE} height={SIZE} viewBox={`0 0 ${SIZE} ${SIZE}`}>
          <circle cx={SIZE / 2} cy={SIZE / 2} r={RADIUS} fill="none" stroke="var(--border)" strokeWidth={STROKE} />
          {segments.map((seg, i) => {
            const fraction = seg.value / safeTotal;
            const length = Math.max(fraction * CIRCUMFERENCE - GAP, 0);
            const offset = offsets[i];
            return (
              <circle
                key={seg.label}
                cx={SIZE / 2}
                cy={SIZE / 2}
                r={RADIUS}
                fill="none"
                strokeWidth={STROKE}
                strokeDasharray={`${length} ${CIRCUMFERENCE - length}`}
                strokeDashoffset={-offset}
                transform={`rotate(-90 ${SIZE / 2} ${SIZE / 2})`}
                className={cn(seg.colorClass, "transition-opacity", hovered !== null && hovered !== i && "opacity-40")}
                stroke="currentColor"
                strokeLinecap="butt"
                onMouseEnter={() => setHovered(i)}
                onMouseLeave={() => setHovered(null)}
              />
            );
          })}
        </svg>
        {/* pointer-events-none: this overlay spans the whole circle (not
            just the hole), so without it the ring's hover handlers below
            never fire — the div always wins hit-testing first. */}
        <div className="pointer-events-none absolute inset-0 flex flex-col items-center justify-center text-center">
          <span className="text-2xl font-semibold tracking-tight">{active ? active.value : total}</span>
          <span className="max-w-24 text-[11px] leading-tight text-muted-foreground">
            {active ? active.label : centerLabel}
          </span>
        </div>
      </div>

      <ul className="flex-1 space-y-2 text-sm">
        {segments.map((seg, i) => {
          const pct = safeTotal ? Math.round((seg.value / safeTotal) * 1000) / 10 : 0;
          return (
            <li
              key={seg.label}
              className={cn(
                "flex items-center gap-2 rounded-md px-1.5 py-1 transition-colors",
                hovered === i && "bg-muted/60",
              )}
              onMouseEnter={() => setHovered(i)}
              onMouseLeave={() => setHovered(null)}
            >
              <span className={cn("size-2.5 shrink-0 rounded-full", seg.dotClass)} aria-hidden />
              <span className="text-muted-foreground">{seg.label}</span>
              <span className="ml-auto font-semibold tabular-nums">{seg.value}</span>
              <span className="w-14 text-right text-xs text-muted-foreground tabular-nums">({pct}%)</span>
            </li>
          );
        })}
      </ul>
    </div>
  );
}
