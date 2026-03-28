"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";

type DateFilter = "today" | "7d" | "30d" | "all";

const DATE_FILTERS: { key: DateFilter; label: string }[] = [
  { key: "today", label: "Today" },
  { key: "7d", label: "7 Days" },
  { key: "30d", label: "30 Days" },
  { key: "all", label: "All Time" },
];

const MOCK_DATA = {
  totalRequests: 1_284,
  successCount: 1_241,
  failureCount: 43,
  totalTokens: 2_847_320,
  inputTokens: 1_423_660,
  outputTokens: 1_423_660,
};

const MOCK_KEYS = [
  { name: "Development", requests: 847, tokens: 1_847_320, success: 820, failure: 27 },
  { name: "Production", requests: 437, tokens: 1_000_000, success: 421, failure: 16 },
];

export default function UsagePage() {
  const [activeFilter, setActiveFilter] = useState<DateFilter>("7d");

  return (
    <div className="space-y-4">
      <section className="rounded-lg border border-slate-700/70 bg-slate-900/40 p-4">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-xl font-semibold tracking-tight text-slate-100">Usage Statistics</h1>
            <div className="mt-1 flex items-center gap-2">
              <div className="h-2 w-2 rounded-full bg-emerald-500" />
              <p className="text-xs text-slate-400">Last synced: just now</p>
            </div>
          </div>
          <Button onClick={() => {}} className="px-2.5 py-1 text-xs">Refresh</Button>
        </div>
      </section>

      <div className="flex flex-wrap gap-1">
        {DATE_FILTERS.map((f) => (
          <Button
            key={f.key}
            variant={activeFilter === f.key ? "secondary" : "ghost"}
            onClick={() => setActiveFilter(f.key)}
            className="px-2.5 py-1 text-xs"
          >{f.label}</Button>
        ))}
      </div>

      <div className="grid grid-cols-[repeat(auto-fit,minmax(150px,1fr))] gap-2">
        {[
          { label: "Total Requests", value: MOCK_DATA.totalRequests.toLocaleString(), tone: "text-slate-100" },
          { label: "Successful", value: MOCK_DATA.successCount.toLocaleString(), tone: "text-emerald-300" },
          { label: "Failed", value: MOCK_DATA.failureCount.toLocaleString(), tone: "text-rose-300" },
          { label: "Total Tokens", value: MOCK_DATA.totalTokens.toLocaleString(), tone: "text-slate-100" },
          { label: "Input Tokens", value: MOCK_DATA.inputTokens.toLocaleString(), tone: "text-slate-100" },
          { label: "Output Tokens", value: MOCK_DATA.outputTokens.toLocaleString(), tone: "text-slate-100" },
        ].map((stat) => (
          <div key={stat.label} className="rounded-lg border border-slate-700/70 bg-slate-900/40 px-2.5 py-2">
            <p className="text-[10px] font-semibold uppercase tracking-[0.08em] text-slate-500">{stat.label}</p>
            <p className={`mt-0.5 text-xs font-semibold ${stat.tone}`}>{stat.value}</p>
          </div>
        ))}
      </div>

      <section className="overflow-x-auto rounded-lg border border-slate-700/70 bg-slate-900/40">
        <div className="border-b border-slate-700/70 px-3 py-2">
          <h2 className="text-[10px] font-semibold uppercase tracking-[0.08em] text-slate-400">Usage by API Key</h2>
        </div>
        <table className="min-w-[500px] w-full text-sm">
          <thead>
            <tr className="border-b border-slate-700/70 bg-slate-900/50">
              <th className="px-3 py-2 text-left text-[10px] font-semibold uppercase tracking-[0.08em] text-slate-400">Key</th>
              <th className="px-3 py-2 text-left text-[10px] font-semibold uppercase tracking-[0.08em] text-slate-400">Requests</th>
              <th className="px-3 py-2 text-left text-[10px] font-semibold uppercase tracking-[0.08em] text-slate-400">Tokens</th>
              <th className="px-3 py-2 text-left text-[10px] font-semibold uppercase tracking-[0.08em] text-slate-400">Success</th>
              <th className="px-3 py-2 text-left text-[10px] font-semibold uppercase tracking-[0.08em] text-slate-400">Failed</th>
            </tr>
          </thead>
          <tbody>
            {MOCK_KEYS.map((k) => (
              <tr key={k.name} className="border-b border-slate-700/60 last:border-b-0 hover:bg-slate-800/30 transition-colors">
                <td className="px-3 py-2 text-xs font-medium text-slate-100">{k.name}</td>
                <td className="px-3 py-2 text-xs text-slate-300">{k.requests.toLocaleString()}</td>
                <td className="px-3 py-2 text-xs text-slate-300">{k.tokens.toLocaleString()}</td>
                <td className="px-3 py-2 text-xs text-emerald-300">{k.success.toLocaleString()}</td>
                <td className="px-3 py-2 text-xs text-rose-300">{k.failure.toLocaleString()}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </section>
    </div>
  );
}
