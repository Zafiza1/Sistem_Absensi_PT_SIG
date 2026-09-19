"use client";

import { usePathname } from "next/navigation";
import type { LucideIcon } from "lucide-react";

import { cn } from "@/lib/utils";
import { NAV_ITEMS, type NavTone } from "@/components/layout/nav-items";

const TONE_CLASSES: Record<NavTone, string> = {
  blue: "bg-primary/10 text-primary dark:bg-primary/20 dark:text-sky-400",
  violet: "bg-violet-50 text-violet-600 dark:bg-violet-500/10 dark:text-violet-400",
  amber: "bg-amber-50 text-amber-600 dark:bg-amber-500/10 dark:text-amber-400",
  cyan: "bg-cyan-50 text-cyan-600 dark:bg-cyan-500/10 dark:text-cyan-400",
  green: "bg-emerald-50 text-emerald-600 dark:bg-emerald-500/10 dark:text-emerald-400",
  rose: "bg-rose-50 text-rose-600 dark:bg-rose-500/10 dark:text-rose-400",
};

export function PageHeader({
  title,
  description,
  action,
  icon,
  tone,
}: {
  title: string;
  description?: string;
  action?: React.ReactNode;
  // Both default to a lookup against the current route in NAV_ITEMS, so
  // most pages don't need to pass either — only override for a header
  // that doesn't map 1:1 to a sidebar route.
  icon?: LucideIcon;
  tone?: NavTone;
}) {
  const pathname = usePathname();
  const current = NAV_ITEMS.find((item) => pathname === item.href || pathname.startsWith(`${item.href}/`));
  const Icon = icon ?? current?.icon;
  const resolvedTone = tone ?? current?.tone ?? "blue";

  return (
    <div className="mb-6 flex flex-wrap items-center justify-between gap-3">
      <div className="flex items-center gap-3.5">
        {Icon && (
          <div
            className={cn(
              "flex size-11 shrink-0 items-center justify-center rounded-xl",
              TONE_CLASSES[resolvedTone],
            )}
            aria-hidden
          >
            <Icon className="size-5.5" />
          </div>
        )}
        <div>
          <h1 className="text-2xl font-semibold tracking-tight text-foreground">{title}</h1>
          {description && <p className="text-sm text-muted-foreground">{description}</p>}
        </div>
      </div>
      {action}
    </div>
  );
}
