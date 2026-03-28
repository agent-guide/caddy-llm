"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import { HelpTooltip } from "@/components/ui/tooltip";

const MOCK_ACCOUNTS = [
  { provider: "anthropic", email: "user@example.com", supported: true, capacity: 0.78, resetTime: "2026-03-29T00:00:00Z" },
  { provider: "openai", email: "user@example.com", supported: true, capacity: 0.45, resetTime: "2026-03-29T00:00:00Z" },
  { provider: "codex", email: "user2@example.com", supported: true, capacity: 0.12, resetTime: "2026-03-28T05:00:00Z" },
];

function CapacityBar({ value }: { value: number }) {
  const pct = Math.round(value * 100);
  const color = pct > 50 ? "bg-emerald-500" : pct > 20 ? "bg-amber-500" : "bg-rose-500";
  return (
    <div className="flex items-center gap-2">
      <div className="flex-1 h-1.5 rounded-full bg-slate-700/50">
        <div className={`h-1.5 rounded-full ${color} transition-all`} style={{ width: `${pct}%` }} />
      </div>
      <span className={`text-xs font-medium w-8 ${pct > 50 ? "text-emerald-300" : pct > 20 ? "text-amber-300" : "text-rose-300"}`}>{pct}%</span>
    </div>
  );
}

export default function QuotaPage() {
  const [loading, setLoading] = useState(false);

  const activeAccounts = MOCK_ACCOUNTS.filter((a) => a.supported).length;
  const overallCapacity = MOCK_ACCOUNTS.reduce((s, a) => s + a.capacity, 0) / MOCK_ACCOUNTS.length;
  const lowCapacityCount = MOCK_ACCOUNTS.filter((a) => a.capacity < 0.2).length;

  return (
    <div className="space-y-4">
      <section className="rounded-lg border border-slate-700/70 bg-slate-900/40 p-4">
        <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
          <div>
            <h1 className="text-xl font-semibold tracking-tight text-slate-100">Quota</h1>
            <p className="mt-1 text-sm text-slate-400">Monitor OAuth account quotas and usage windows.</p>
          </div>
          <Button onClick={() => setLoading(false)} disabled={loading} className="px-2.5 py-1 text-xs">
            {loading ? "Loading..." : "Refresh"}
          </Button>
        </div>
      </section>

      <section className="grid grid-cols-2 gap-2 lg:grid-cols-4">
        {[
          { label: "Active Accounts", value: activeAccounts },
          { label: "Overall Capacity", value: `${Math.round(overallCapacity * 100)}%`, tooltip: "Weighted average of remaining quota" },
          { label: "Low Capacity", value: lowCapacityCount, tooltip: "Accounts with quota below 20%" },
          { label: "Providers", value: new Set(MOCK_ACCOUNTS.map((a) => a.provider)).size },
        ].map((stat) => (
          <div key={stat.label} className="rounded-lg border border-slate-700/70 bg-slate-900/40 px-2.5 py-2">
            <p className="text-[10px] font-semibold uppercase tracking-[0.08em] text-slate-500">
              {stat.label}
              {stat.tooltip && <HelpTooltip content={stat.tooltip} />}
            </p>
            <p className="mt-0.5 text-xs font-semibold text-slate-100">{stat.value}</p>
          </div>
        ))}
      </section>

      <section className="space-y-2">
        {MOCK_ACCOUNTS.map((account, i) => (
          <div key={i} className="rounded-lg border border-slate-700/70 bg-slate-900/40 p-4">
            <div className="flex items-center justify-between mb-3">
              <div>
                <span className="text-sm font-semibold capitalize text-slate-100">{account.provider}</span>
                <span className="ml-2 text-xs text-slate-400">{account.email}</span>
              </div>
              <span className={`inline-flex items-center rounded-sm px-1.5 py-0.5 text-[10px] font-medium ${account.supported ? "bg-emerald-500/15 text-emerald-300" : "bg-slate-700/40 text-slate-400"}`}>
                {account.supported ? "active" : "inactive"}
              </span>
            </div>
            <div className="space-y-1.5">
              <div className="flex items-center justify-between text-xs text-slate-400 mb-1">
                <span>Remaining capacity</span>
                <span>Resets {new Date(account.resetTime).toLocaleDateString()}</span>
              </div>
              <CapacityBar value={account.capacity} />
            </div>
          </div>
        ))}
      </section>
    </div>
  );
}
