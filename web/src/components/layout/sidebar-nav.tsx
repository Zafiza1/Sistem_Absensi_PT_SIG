"use client";

import Link from "next/link";
import Image from "next/image";
import { usePathname } from "next/navigation";

import { cn } from "@/lib/utils";
import { useAuth } from "@/lib/auth-context";
import { NAV_ITEMS } from "./nav-items";

export function SidebarNav({ onNavigate }: { onNavigate?: () => void }) {
  const pathname = usePathname();
  const { user } = useAuth();

  const items = NAV_ITEMS.filter((item) => !item.roles || (user && item.roles.includes(user.role)));

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

      <nav className="flex-1 space-y-0.5 overflow-y-auto px-3 pb-4">
        {items.map((item) => {
          const active = pathname === item.href || pathname.startsWith(`${item.href}/`);
          const Icon = item.icon;
          return (
            <Link
              key={item.href}
              href={item.href}
              onClick={onNavigate}
              aria-current={active ? "page" : undefined}
              className={cn(
                "group relative flex items-center gap-3 rounded-md py-2 pr-3 pl-3.5 text-sm font-medium transition-colors",
                active
                  ? "bg-sidebar-accent text-white"
                  : "text-sidebar-foreground/80 hover:bg-sidebar-accent/60 hover:text-white",
              )}
            >
              <span
                className={cn(
                  "absolute top-1/2 left-0 h-4.5 w-[3px] -translate-y-1/2 rounded-full bg-sidebar-primary transition-opacity",
                  active ? "opacity-100" : "opacity-0",
                )}
                aria-hidden
              />
              <Icon className={cn("size-4.5 shrink-0", active && "text-sidebar-primary")} />
              {item.label}
            </Link>
          );
        })}
      </nav>
    </div>
  );
}
