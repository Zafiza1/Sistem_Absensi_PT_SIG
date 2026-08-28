"use client";

import { ShieldAlert } from "lucide-react";

import { useAuth } from "@/lib/auth-context";
import type { Role } from "@/lib/types";

/**
 * Client-side backup to the backend's own RBAC (which is the real
 * enforcement — see backend/README.md's SUPER_ADMIN-only routes). This
 * only avoids a confusing raw 403 mid-page for a user who can see the nav
 * link exists but shouldn't be here — the API would still refuse the
 * requests either way.
 */
export function RequireRole({ roles, children }: { roles: Role[]; children: React.ReactNode }) {
  const { user } = useAuth();

  if (!user || !roles.includes(user.role)) {
    return (
      <div className="flex min-h-[50vh] flex-col items-center justify-center gap-2 text-center">
        <ShieldAlert className="size-10 text-muted-foreground" />
        <p className="font-medium">Anda tidak memiliki akses ke halaman ini</p>
        <p className="text-sm text-muted-foreground">Hubungi Super Admin jika Anda memerlukan akses.</p>
      </div>
    );
  }

  return <>{children}</>;
}
