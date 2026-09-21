"use client";

import { useState } from "react";

import { Skeleton } from "@/components/ui/skeleton";

export interface TrendPoint {
  label: string; // short x-axis label, e.g. "16 Mei"
  hadir: number;
  terlambat: number;
  tidakHadir: number;
}

const SERIES = [
  { key: "hadir" as const, label: "Hadir", stroke: "#10b981", text: "text-emerald-600" },
  { key: "terlambat" as const, label: "Terlambat", stroke: "#f59e0b", text: "text-amber-600" },
  { key: "tidakHadir" as const, label: "Tidak Hadir", stroke: "#dc2626", text: "text-red-600" },
];

const WIDTH = 640;
const HEIGHT = 240;
const PAD = { top: 28, right: 12, bottom: 24, left: 30 };
const PLOT_W = WIDTH - PAD.left - PAD.right;
const PLOT_H = HEIGHT - PAD.top - PAD.bottom;

const Y_TICKS = 4;

// Picks an integer tick step (>= 1) so the four gridlines are always
// distinct after rounding — a fractional step (e.g. max 3 / 4 ticks =
// 0.75) rounds two adjacent ticks to the same displayed integer.
function tickStep(maxValue: number): number {
  return Math.max(1, Math.ceil(maxValue / Y_TICKS));
}

export function AttendanceTrendChart({ data, loading }: { data: TrendPoint[]; loading: boolean }) {
  const [hoverIndex, setHoverIndex] = useState<number | null>(null);

  if (loading) {
    return <Skeleton className="h-56 w-full" />;
  }

  if (data.length === 0) {
    return (
      <div className="flex h-56 items-center justify-center text-sm text-muted-foreground">
        Belum ada data kehadiran
      </div>
    );
  }

  const rawMax = Math.max(1, ...data.flatMap((d) => [d.hadir, d.terlambat, d.tidakHadir]));
  const step = tickStep(rawMax);
  const maxValue = step * Y_TICKS;
  const xStep = data.length > 1 ? PLOT_W / (data.length - 1) : 0;

  const xAt = (i: number) => PAD.left + i * xStep;
  const yAt = (v: number) => PAD.top + PLOT_H - (v / maxValue) * PLOT_H;

  const linePath = (key: "hadir" | "terlambat" | "tidakHadir") =>
    data.map((d, i) => `${i === 0 ? "M" : "L"}${xAt(i)},${yAt(d[key])}`).join(" ");

  const hovered = hoverIndex !== null ? data[hoverIndex] : null;

  return (
    <div className="space-y-2">
      <div className="flex flex-wrap items-center gap-4">
        {SERIES.map((s) => (
          <span key={s.key} className="flex items-center gap-1.5 text-xs text-muted-foreground">
            <span className="size-2 shrink-0 rounded-full" style={{ backgroundColor: s.stroke }} aria-hidden />
            {s.label}
          </span>
        ))}
      </div>

      <div className="relative">
        <svg
          viewBox={`0 0 ${WIDTH} ${HEIGHT}`}
          className="h-56 w-full"
          role="img"
          aria-label="Grafik kehadiran 7 hari terakhir"
        >
          {/* Recessive horizontal gridlines */}
          {Array.from({ length: Y_TICKS + 1 }).map((_, i) => {
            const v = step * i;
            const y = yAt(v);
            return (
              <g key={i}>
                <line x1={PAD.left} x2={WIDTH - PAD.right} y1={y} y2={y} stroke="var(--border)" strokeWidth={1} />
                <text x={PAD.left - 8} y={y} textAnchor="end" dominantBaseline="middle" className="fill-muted-foreground text-[9px]">
                  {v}
                </text>
              </g>
            );
          })}

          {/* Hover crosshair */}
          {hoverIndex !== null && (
            <line
              x1={xAt(hoverIndex)}
              x2={xAt(hoverIndex)}
              y1={PAD.top}
              y2={PAD.top + PLOT_H}
              stroke="var(--border)"
              strokeWidth={1}
              strokeDasharray="3 3"
            />
          )}

          {/* Series lines + markers */}
          {SERIES.map((s) => (
            <g key={s.key}>
              <path d={linePath(s.key)} fill="none" stroke={s.stroke} strokeWidth={2} strokeLinecap="round" strokeLinejoin="round" />
              {data.map((d, i) => (
                <circle
                  key={i}
                  cx={xAt(i)}
                  cy={yAt(d[s.key])}
                  r={hoverIndex === i ? 4.5 : 3}
                  fill={s.stroke}
                  stroke="var(--card)"
                  strokeWidth={2}
                />
              ))}
              {/* Direct labels on the primary (Hadir) series only — the
                  other two stay legend + hover-only to avoid tripling the
                  label count on a 7-point chart. */}
              {s.key === "hadir" &&
                data.map((d, i) => (
                  <text
                    key={i}
                    x={xAt(i)}
                    y={yAt(d.hadir) - 10}
                    textAnchor="middle"
                    className="fill-foreground text-[10px] font-semibold"
                  >
                    {d.hadir}
                  </text>
                ))}
            </g>
          ))}

          {/* X-axis labels */}
          {data.map((d, i) => (
            <text key={i} x={xAt(i)} y={HEIGHT - 4} textAnchor="middle" className="fill-muted-foreground text-[9px]">
              {d.label}
            </text>
          ))}

          {/* Invisible hit targets, one per data point, wider than the mark */}
          {data.map((d, i) => (
            <rect
              key={i}
              x={xAt(i) - xStep / 2}
              y={0}
              width={xStep || WIDTH}
              height={HEIGHT}
              fill="transparent"
              onMouseEnter={() => setHoverIndex(i)}
              onMouseLeave={() => setHoverIndex(null)}
            />
          ))}
        </svg>

        {hovered && hoverIndex !== null && (
          <div
            className="pointer-events-none absolute top-1 z-10 -translate-x-1/2 rounded-lg border bg-popover px-2.5 py-1.5 text-xs shadow-md"
            style={{ left: `${(xAt(hoverIndex) / WIDTH) * 100}%` }}
          >
            <p className="mb-1 font-medium">{hovered.label}</p>
            {SERIES.map((s) => (
              <p key={s.key} className={s.text}>
                {s.label}: <span className="font-semibold">{hovered[s.key]}</span>
              </p>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
