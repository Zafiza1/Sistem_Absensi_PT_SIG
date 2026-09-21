"use client";

import { useEffect, useState } from "react";
import { Clock } from "lucide-react";

// Ticks on its own so a per-second re-render never cascades into the rest
// of the dashboard (attendance list, charts, etc.) re-rendering with it.
export function LiveClock() {
  const [now, setNow] = useState<Date | null>(null);

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect
    setNow(new Date());
    const id = setInterval(() => setNow(new Date()), 1000);
    return () => clearInterval(id);
  }, []);

  return (
    <div className="flex items-center gap-3 rounded-lg bg-muted/60 px-4 py-3">
      <div className="flex size-9 shrink-0 items-center justify-center rounded-lg bg-primary/10 text-primary">
        <Clock className="size-4.5" />
      </div>
      <div className="min-w-0">
        <p className="text-xs text-muted-foreground">Waktu Sekarang</p>
        <p className="font-mono text-lg leading-tight font-semibold tracking-tight tabular-nums">
          {now
            ? `${now.toLocaleTimeString("id-ID", { hour: "2-digit", minute: "2-digit", second: "2-digit", timeZone: "Asia/Jakarta" })} WIB`
            : "--:--:-- WIB"}
        </p>
      </div>
    </div>
  );
}
