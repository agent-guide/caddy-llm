"use client";

import { useState, useEffect } from "react";
import { DashboardHeader } from "@/components/dashboard-header";
import { UserPanel } from "@/components/user-panel";
import { getUsername } from "@/lib/auth";

export function DashboardShell({ children }: { children: React.ReactNode }) {
  const [panelOpen, setPanelOpen] = useState(false);
  const [username, setUsername] = useState("admin");

  useEffect(() => {
    const u = getUsername();
    if (u) setUsername(u);
  }, []);

  return (
    <>
      <DashboardHeader
        username={username}
        isAdmin={true}
        onUserClick={() => setPanelOpen(true)}
      />
      {children}
      <UserPanel
        isOpen={panelOpen}
        onClose={() => setPanelOpen(false)}
        username={username}
        isAdmin={true}
      />
    </>
  );
}
