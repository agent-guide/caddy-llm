"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import { useToast } from "@/components/ui/toast";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";

interface MemoryEntry {
  id: string;
  key: string;
  size: number;
  createdAt: string;
  expiresAt: string | null;
}

const MOCK_ENTRIES: MemoryEntry[] = [
  { id: "1", key: "session:user1:context", size: 4096, createdAt: "2026-03-28T08:00:00Z", expiresAt: "2026-03-29T08:00:00Z" },
  { id: "2", key: "session:user2:context", size: 2048, createdAt: "2026-03-28T10:00:00Z", expiresAt: null },
  { id: "3", key: "agent:code-assistant:state", size: 8192, createdAt: "2026-03-27T12:00:00Z", expiresAt: null },
];

export default function MemoryPage() {
  const [entries, setEntries] = useState<MemoryEntry[]>(MOCK_ENTRIES);
  const [showConfirm, setShowConfirm] = useState(false);
  const { showToast } = useToast();

  const totalSize = entries.reduce((s, e) => s + e.size, 0);

  return (
    <div className="space-y-4">
      <section className="rounded-lg border border-slate-700/70 bg-slate-900/40 p-4">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-xl font-semibold tracking-tight text-slate-100">Memory</h1>
            <p className="mt-1 text-sm text-slate-400">Manage conversation memory and agent state storage.</p>
          </div>
          <Button variant="danger" onClick={() => setShowConfirm(true)} className="px-2.5 py-1 text-xs">Clear All</Button>
        </div>
      </section>

      <div className="grid grid-cols-2 gap-2 lg:grid-cols-3">
        {[
          { label: "Total Entries", value: entries.length },
          { label: "Total Size", value: `${(totalSize / 1024).toFixed(1)} KB` },
          { label: "With Expiry", value: entries.filter((e) => e.expiresAt).length },
        ].map((stat) => (
          <div key={stat.label} className="rounded-lg border border-slate-700/70 bg-slate-900/40 px-2.5 py-2">
            <p className="text-[10px] font-semibold uppercase tracking-[0.08em] text-slate-500">{stat.label}</p>
            <p className="mt-0.5 text-xs font-semibold text-slate-100">{stat.value}</p>
          </div>
        ))}
      </div>

      <section className="overflow-x-auto rounded-lg border border-slate-700/70 bg-slate-900/40">
        <table className="min-w-[600px] w-full text-sm">
          <thead>
            <tr className="border-b border-slate-700/70 bg-slate-900/95 backdrop-blur-sm">
              {["Key", "Size", "Created", "Expires", "Actions"].map((h) => (
                <th key={h} className="px-3 py-2 text-left text-[10px] font-semibold uppercase tracking-[0.08em] text-slate-400">{h}</th>
              ))}
            </tr>
          </thead>
          <tbody>
            {entries.map((entry) => (
              <tr key={entry.id} className="border-b border-slate-700/60 last:border-b-0 hover:bg-slate-800/30 transition-colors">
                <td className="px-3 py-2 font-mono text-xs text-slate-100 max-w-xs truncate">{entry.key}</td>
                <td className="px-3 py-2 text-xs text-slate-400">{entry.size} B</td>
                <td className="px-3 py-2 text-xs text-slate-400">{new Date(entry.createdAt).toLocaleString()}</td>
                <td className="px-3 py-2 text-xs text-slate-400">{entry.expiresAt ? new Date(entry.expiresAt).toLocaleString() : "Never"}</td>
                <td className="px-3 py-2">
                  <Button variant="danger" onClick={() => { setEntries((prev) => prev.filter((e) => e.id !== entry.id)); showToast("Entry deleted", "success"); }} className="px-2 py-1 text-xs">Delete</Button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </section>

      <ConfirmDialog
        isOpen={showConfirm}
        onClose={() => setShowConfirm(false)}
        onConfirm={() => { setEntries([]); showToast("Memory cleared", "success"); }}
        title="Clear All Memory"
        message="Are you sure you want to clear all memory entries? This cannot be undone."
        confirmLabel="Clear All"
        variant="danger"
      />
    </div>
  );
}
