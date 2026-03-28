"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";

const MOCK_LOGS = [
  { id: 1, time: Date.now() - 5000, level: "info", msg: "LLM API request processed successfully" },
  { id: 2, time: Date.now() - 15000, level: "info", msg: "Provider openai: model gpt-4o selected" },
  { id: 3, time: Date.now() - 30000, level: "warn", msg: "Rate limit approaching for provider anthropic" },
  { id: 4, time: Date.now() - 60000, level: "error", msg: "Request to ollama failed: connection refused" },
  { id: 5, time: Date.now() - 120000, level: "info", msg: "API key validated successfully" },
];

const LEVEL_COLORS: Record<string, string> = {
  error: "text-red-400 bg-red-500/10 border-red-500/30",
  warn: "text-yellow-400 bg-yellow-500/10 border-yellow-500/30",
  info: "text-blue-400 bg-blue-500/10 border-blue-500/30",
  debug: "text-gray-400 bg-gray-500/10 border-gray-500/30",
};

function formatRelativeTime(timestamp: number): string {
  const diff = Date.now() - timestamp;
  if (diff < 60000) return `${Math.floor(diff / 1000)}s ago`;
  if (diff < 3600000) return `${Math.floor(diff / 60000)}m ago`;
  return `${Math.floor(diff / 3600000)}h ago`;
}

export default function MonitoringPage() {
  const [showConfirm, setShowConfirm] = useState(false);
  const [logs] = useState(MOCK_LOGS);

  return (
    <div className="space-y-4">
      <section className="rounded-lg border border-slate-700/70 bg-slate-900/40 p-4">
        <h1 className="text-xl font-semibold tracking-tight text-slate-100">Monitoring</h1>
      </section>

      <section className="grid grid-cols-2 gap-2 lg:grid-cols-4">
        {[
          { label: "Service Status", value: "Online", tone: "text-emerald-400" },
          { label: "Uptime", value: "2d 4h 12m", tone: "text-slate-100" },
          { label: "Requests/min", value: "42", tone: "text-slate-100" },
          { label: "Avg Latency", value: "320 ms", tone: "text-cyan-300" },
        ].map((stat) => (
          <div key={stat.label} className="rounded-lg border border-slate-700/70 bg-slate-900/40 px-2.5 py-2">
            <p className="text-[10px] font-semibold uppercase tracking-[0.08em] text-slate-500">{stat.label}</p>
            <p className={`mt-0.5 text-xs font-semibold ${stat.tone}`}>{stat.value}</p>
          </div>
        ))}
      </section>

      <section className="rounded-lg border border-slate-700/70 bg-slate-900/40 p-4 space-y-3">
        <div className="flex items-center justify-between">
          <h2 className="text-sm font-semibold text-slate-100">Service Control</h2>
        </div>
        <div className="flex flex-wrap gap-2">
          <Button variant="secondary" className="px-2.5 py-1 text-xs">Reload Config</Button>
          <Button variant="danger" onClick={() => setShowConfirm(true)} className="px-2.5 py-1 text-xs">Restart Service</Button>
        </div>
      </section>

      <section className="overflow-hidden rounded-lg border border-slate-700/70 bg-slate-900/40">
        <div className="flex items-center justify-between border-b border-slate-700/70 bg-slate-900/50 px-3 py-2">
          <span className="text-[10px] font-semibold uppercase tracking-[0.08em] text-slate-400">Live Logs</span>
          <div className="flex items-center gap-2">
            <div className="h-1.5 w-1.5 rounded-full bg-emerald-500 animate-pulse-dot" />
            <span className="text-xs text-slate-400">Live</span>
          </div>
        </div>
        <div className="max-h-80 overflow-auto">
          {logs.map((log) => (
            <div key={log.id} className="flex items-start gap-3 border-b border-slate-700/60 px-3 py-2 last:border-b-0 hover:bg-slate-800/20 transition-colors">
              <span className={`inline-flex shrink-0 items-center rounded-full border px-2 py-0.5 text-[10px] font-medium ${LEVEL_COLORS[log.level] ?? LEVEL_COLORS.debug}`}>
                {log.level.toUpperCase()}
              </span>
              <span className="flex-1 text-xs text-slate-300 font-mono">{log.msg}</span>
              <span className="shrink-0 text-[10px] text-slate-500">{formatRelativeTime(log.time)}</span>
            </div>
          ))}
        </div>
      </section>

      <ConfirmDialog
        isOpen={showConfirm}
        onClose={() => setShowConfirm(false)}
        onConfirm={() => {}}
        title="Restart Service"
        message="Are you sure you want to restart the agent gateway service?"
        confirmLabel="Restart"
        variant="warning"
      />
    </div>
  );
}
