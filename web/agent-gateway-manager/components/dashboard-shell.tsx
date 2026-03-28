"use client";

import { useState } from "react";
import { DashboardHeader } from "@/components/dashboard-header";
import { UserPanel } from "@/components/user-panel";

// Stub user — replace with real auth context when backend is ready
const STUB_USER = { username: "admin", isAdmin: true };

export function DashboardShell({ children }: { children: React.ReactNode }) {
  const [panelOpen, setPanelOpen] = useState(false);
  const user = STUB_USER;

  return (
    <>
      <DashboardHeader
        username={user.username}
        isAdmin={user.isAdmin}
        onUserClick={() => setPanelOpen(true)}
      />
      {children}
      <UserPanel
        isOpen={panelOpen}
        onClose={() => setPanelOpen(false)}
        username={user.username}
        isAdmin={user.isAdmin}
      />
    </>
  );
}
