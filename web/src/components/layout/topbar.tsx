"use client";

import { useState } from "react";
import { usePathname } from "next/navigation";
import { Menu, LogOut, User as UserIcon } from "lucide-react";

import { useAuth } from "@/lib/auth-context";
import { ROLE_LABELS } from "@/lib/types";
import { Button } from "@/components/ui/button";
import { Sheet, SheetContent, SheetTitle } from "@/components/ui/sheet";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { SidebarNav } from "./sidebar-nav";
import { NAV_ITEMS } from "./nav-items";

function initials(name: string): string {
  const parts = name.trim().split(/\s+/);
  if (parts.length === 0 || !parts[0]) return "?";
  if (parts.length === 1) return parts[0].slice(0, 1).toUpperCase();
  return (parts[0][0] + parts[parts.length - 1][0]).toUpperCase();
}

export function Topbar() {
  const { user, logout } = useAuth();
  const [mobileNavOpen, setMobileNavOpen] = useState(false);
  const pathname = usePathname();

  const currentLabel = NAV_ITEMS.find(
    (item) => pathname === item.href || pathname.startsWith(`${item.href}/`),
  )?.label;

  return (
    <header className="flex h-14 shrink-0 items-center gap-3 border-b bg-background px-4 md:px-6">
      <Sheet open={mobileNavOpen} onOpenChange={setMobileNavOpen}>
        <SheetContent side="left" className="w-64 border-none p-0">
          <SheetTitle className="sr-only">Menu navigasi</SheetTitle>
          <SidebarNav onNavigate={() => setMobileNavOpen(false)} />
        </SheetContent>
      </Sheet>
      <Button variant="ghost" size="icon" className="md:hidden" onClick={() => setMobileNavOpen(true)}>
        <Menu className="size-5" />
        <span className="sr-only">Buka menu</span>
      </Button>

      {currentLabel && <h2 className="hidden text-sm font-semibold text-foreground md:block">{currentLabel}</h2>}

      <div className="flex-1" />

      {user && (
        <DropdownMenu>
          <DropdownMenuTrigger
            render={<Button variant="ghost" className="flex h-auto items-center gap-2 px-2 py-1.5" />}
          >
            <Avatar className="size-7 border border-border">
              <AvatarFallback className="bg-primary/10 text-xs font-semibold text-primary">
                {initials(user.name)}
              </AvatarFallback>
            </Avatar>
            <span className="hidden text-left leading-tight sm:block">
              <span className="block text-sm font-medium">{user.name}</span>
              <span className="block text-xs text-muted-foreground">{ROLE_LABELS[user.role]}</span>
            </span>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end" className="w-56">
            <DropdownMenuLabel>
              <p className="flex items-center gap-1.5 text-sm font-medium">
                <UserIcon className="size-3.5 text-muted-foreground" />
                {user.name}
              </p>
              <p className="pl-5 text-xs font-normal text-muted-foreground">{user.email}</p>
            </DropdownMenuLabel>
            <DropdownMenuSeparator />
            <DropdownMenuItem variant="destructive" onClick={() => logout()}>
              <LogOut className="size-4" />
              Keluar
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      )}
    </header>
  );
}
