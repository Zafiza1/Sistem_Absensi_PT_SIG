"use client";

import Link from "next/link";
import Image from "next/image";
import { usePathname } from "next/navigation";

import { cn } from "@/lib/utils";
import { useAuth } from "@/lib/auth-context";
import { NAV_ITEMS, type NavItem, type NavTone } from "./nav-items";

// Sidebar surface is permanently dark navy regardless of light/dark theme
// (see --sidebar in globals.css), so these tints are tuned for that one
// background rather than swapping between light/dark variants.
const TONE_CHIP_CLASSES: Record<NavTone, string> = {
  blue: "bg-sky-400/15 text-sky-300",
  violet: "bg-violet-400/15 text-violet-300",
  amber: "bg-amber-400/15 text-amber-300",
  cyan: "bg-cyan-400/15 text-cyan-300",
  green: "bg-emerald-400/15 text-emerald-300",
  rose: "bg-rose-400/15 text-rose-300",
};

function groupBySection(items: NavItem[]): { section: string; items: NavItem[] }[] {
  const groups: { section: string; items: NavItem[] }[] = [];
  for (const item of items) {
    const last = groups[groups.length - 1];
    if (last && last.section === item.section) last.items.push(item);
    else groups.push({ section: item.section, items: [item] });
  }
  return groups;
}

export function SidebarNav({ onNavigate }: { onNavigate?: () => void }) {
  const pathname = usePathname();
  const { user } = useAuth();

  const items = NAV_ITEMS.filter((item) => !item.roles || (user && item.roles.includes(user.role)));
  const groups = groupBySection(items);

  return (
    <div className="flex h-full flex-col bg-sidebar text-sidebar-foreground">
      <div className="flex items-center gap-2.5 px-5 py-5">
        <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-white">
          <Image src="/logo.png" alt="" width={26} height={17} aria-hidden />
        </div>
        <div className="min-w-0 leading-tight">
          <p className="truncate text-sm font-semibold text-white">PT Surya Inti Gas</p>
          <p className="truncate text-xs text-sidebar-foreground/70">Sistem Absensi</p>
        </div>
      </div>

      <nav className="flex-1 space-y-4 overflow-y-auto px-3 pb-4">
        {groups.map((group) => (
          <div key={group.section}>
            {group.section !== "Utama" && (
              <p className="mb-1 px-3 text-[10px] font-semibold tracking-wider text-sidebar-foreground/40 uppercase">
                {group.section}
              </p>
            )}
            <div className="space-y-0.5">
              {group.items.map((item) => {
                const active = pathname === item.href || pathname.startsWith(`${item.href}/`);
                const Icon = item.icon;
                return (
                  <Link
                    key={item.href}
                    href={item.href}
                    onClick={onNavigate}
                    aria-current={active ? "page" : undefined}
                    className={cn(
                      "group flex items-center gap-2.5 rounded-lg py-1.5 pr-3 pl-1.5 text-sm font-medium transition-colors",
                      active
                        ? "bg-sidebar-accent text-white"
                        : "text-sidebar-foreground/80 hover:bg-sidebar-accent/60 hover:text-white",
                    )}
                  >
                    <span
                      className={cn(
                        "flex size-7 shrink-0 items-center justify-center rounded-md transition-colors",
                        active
                          ? TONE_CHIP_CLASSES[item.tone]
                          : "text-sidebar-foreground/60 group-hover:text-white",
                      )}
                    >
                      <Icon className="size-4" />
                    </span>
                    {item.label}
                  </Link>
                );
              })}
            </div>
          </div>
        ))}
      </nav>
    </div>
  );
}
