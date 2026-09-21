"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";

import { cn } from "@/lib/utils";
import { useAuth } from "@/lib/auth-context";
import { SidebarNav } from "@/components/layout/sidebar-nav";
import { Topbar } from "@/components/layout/topbar";

const SIDEBAR_COLLAPSED_KEY = "absensi_sidebar_collapsed";

export default function DashboardLayout({ children }: { children: React.ReactNode }) {
  const { user, loading } = useAuth();
  const router = useRouter();
  // Lazy initializer instead of an effect: this tree only ever renders
  // once `loading` is false (see the early return below), so the server
  // and first client render both stop at "Memuat..." — reading
  // localStorage here can't cause a hydration mismatch.
  const [collapsed, setCollapsed] = useState(() => {
    if (typeof window === "undefined") return false;
    return window.localStorage.getItem(SIDEBAR_COLLAPSED_KEY) === "1";
  });

  useEffect(() => {
    if (!loading && !user) {
      router.replace("/login");
    }
  }, [loading, user, router]);

  function toggleCollapsed() {
    setCollapsed((prev) => {
      const next = !prev;
      window.localStorage.setItem(SIDEBAR_COLLAPSED_KEY, next ? "1" : "0");
      return next;
    });
  }

  // While the initial /auth/me check is in flight, or once it's failed and
  // the redirect above is about to fire, render nothing rather than a
  // flash of dashboard chrome with no data behind it.
  if (loading || !user) {
    return (
      <div className="flex min-h-screen items-center justify-center text-sm text-muted-foreground">
        Memuat...
      </div>
    );
  }

  return (
    <div className="flex min-h-screen">
      <aside
        className={cn(
          "sticky top-0 hidden h-screen shrink-0 overflow-hidden border-sidebar-border transition-[width] duration-200 ease-in-out md:block",
          collapsed ? "w-0 border-r-0" : "w-64 border-r",
        )}
      >
        {/* Fixed-width inner wrapper: the aside animates width down to 0,
            but the nav itself keeps its real width so labels never wrap
            or squish mid-transition — the outer element just clips it. */}
        <div className="h-full w-64">
          <SidebarNav />
        </div>
      </aside>
      <div className="flex min-w-0 flex-1 flex-col">
        <Topbar sidebarCollapsed={collapsed} onToggleSidebar={toggleCollapsed} />
        <main className="flex-1 overflow-x-auto bg-background p-4 md:p-6">{children}</main>
      </div>
    </div>
  );
}
