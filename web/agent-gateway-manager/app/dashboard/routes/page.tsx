"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Modal, ModalHeader, ModalTitle, ModalContent, ModalFooter } from "@/components/ui/modal";
import { useToast } from "@/components/ui/toast";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import { HelpTooltip } from "@/components/ui/tooltip";

type TargetMode = "weighted" | "failover" | "conditional";
type SelectionStrategy = "auto" | "weighted" | "failover" | "conditional";

interface RouteTarget {
  providerRef: string;
  mode: TargetMode;
  weight?: number;
  priority?: number;
  disabled?: boolean;
}

interface Route {
  id: string;
  name: string;
  description: string;
  disabled: boolean;
  targets: RouteTarget[];
  policy: {
    auth: { requireLocalAPIKey: boolean };
    selection: { strategy: SelectionStrategy };
    timeoutSeconds: number;
    retry: { maxAttempts: number };
    allowStreaming?: boolean;
    allowTools?: boolean;
  };
  createdAt: string;
  updatedAt: string;
}

const MOCK_ROUTES: Route[] = [
  {
    id: "route-1",
    name: "default",
    description: "Default route for all LLM requests",
    disabled: false,
    targets: [
      { providerRef: "openai", mode: "weighted", weight: 80 },
      { providerRef: "anthropic", mode: "weighted", weight: 20 },
    ],
    policy: {
      auth: { requireLocalAPIKey: true },
      selection: { strategy: "weighted" },
      timeoutSeconds: 120,
      retry: { maxAttempts: 2 },
      allowStreaming: true,
      allowTools: true,
    },
    createdAt: "2026-01-10",
    updatedAt: "2026-03-15",
  },
  {
    id: "route-2",
    name: "ha-fallback",
    description: "High-availability route with failover",
    disabled: false,
    targets: [
      { providerRef: "openai", mode: "failover", priority: 1 },
      { providerRef: "openrouter", mode: "failover", priority: 2 },
    ],
    policy: {
      auth: { requireLocalAPIKey: true },
      selection: { strategy: "failover" },
      timeoutSeconds: 60,
      retry: { maxAttempts: 3 },
      allowStreaming: true,
    },
    createdAt: "2026-02-20",
    updatedAt: "2026-02-20",
  },
];

const STRATEGY_COLORS: Record<SelectionStrategy, string> = {
  auto: "bg-blue-500/15 text-blue-300",
  weighted: "bg-violet-500/15 text-violet-300",
  failover: "bg-amber-500/15 text-amber-300",
  conditional: "bg-cyan-500/15 text-cyan-300",
};

const MODE_COLORS: Record<TargetMode, string> = {
  weighted: "bg-violet-500/15 text-violet-300",
  failover: "bg-amber-500/15 text-amber-300",
  conditional: "bg-cyan-500/15 text-cyan-300",
};

export default function RoutesPage() {
  const [routes, setRoutes] = useState<Route[]>(MOCK_ROUTES);
  const [expandedId, setExpandedId] = useState<string | null>(null);
  const [isCreateOpen, setIsCreateOpen] = useState(false);
  const [showConfirm, setShowConfirm] = useState(false);
  const [pendingDeleteId, setPendingDeleteId] = useState<string | null>(null);
  const { showToast } = useToast();

  // Create form state
  const [formName, setFormName] = useState("");
  const [formDesc, setFormDesc] = useState("");
  const [formProvider, setFormProvider] = useState("");
  const [formStrategy, setFormStrategy] = useState<SelectionStrategy>("auto");
  const [formRequireKey, setFormRequireKey] = useState(true);
  const [formTimeout, setFormTimeout] = useState("120");

  const activeCount = routes.filter((r) => !r.disabled).length;
  const totalTargets = routes.reduce((s, r) => s + r.targets.length, 0);

  const openCreate = () => {
    setFormName(""); setFormDesc(""); setFormProvider("");
    setFormStrategy("auto"); setFormRequireKey(true); setFormTimeout("120");
    setIsCreateOpen(true);
  };

  const handleCreate = () => {
    if (!formName.trim()) { showToast("Route name is required", "error"); return; }
    const newRoute: Route = {
      id: `route-${Date.now()}`,
      name: formName.trim(),
      description: formDesc.trim(),
      disabled: false,
      targets: formProvider.trim()
        ? [{ providerRef: formProvider.trim(), mode: "weighted", weight: 100 }]
        : [],
      policy: {
        auth: { requireLocalAPIKey: formRequireKey },
        selection: { strategy: formStrategy },
        timeoutSeconds: parseInt(formTimeout, 10) || 120,
        retry: { maxAttempts: 1 },
      },
      createdAt: new Date().toISOString().split("T")[0],
      updatedAt: new Date().toISOString().split("T")[0],
    };
    setRoutes((prev) => [...prev, newRoute]);
    setIsCreateOpen(false);
    showToast("Route created", "success");
  };

  const handleToggleDisabled = (id: string) => {
    setRoutes((prev) =>
      prev.map((r) => r.id === id ? { ...r, disabled: !r.disabled } : r)
    );
  };

  const handleDelete = () => {
    if (!pendingDeleteId) return;
    setRoutes((prev) => prev.filter((r) => r.id !== pendingDeleteId));
    setExpandedId(null);
    showToast("Route deleted", "success");
  };

  return (
    <div className="space-y-4">
      {/* Header */}
      <section className="rounded-lg border border-slate-700/70 bg-slate-900/40 p-4">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-xl font-semibold tracking-tight text-slate-100">Routes</h1>
            <p className="mt-1 text-xs text-slate-400">
              Define routing rules that map incoming requests to upstream LLM providers.
              <HelpTooltip content="Each route owns target selection, auth policy, rate limits, and retry behavior." />
            </p>
          </div>
          <Button onClick={openCreate} className="px-2.5 py-1 text-xs">
            Create Route
          </Button>
        </div>
      </section>

      {/* Stats */}
      <section className="grid grid-cols-2 gap-2 lg:grid-cols-4">
        {[
          { label: "Total Routes", value: routes.length },
          { label: "Active", value: activeCount },
          { label: "Disabled", value: routes.length - activeCount },
          { label: "Total Targets", value: totalTargets },
        ].map((stat) => (
          <div key={stat.label} className="rounded-lg border border-slate-700/70 bg-slate-900/40 px-2.5 py-2">
            <p className="text-[10px] font-semibold uppercase tracking-[0.12em] text-slate-500">{stat.label}</p>
            <p className="mt-0.5 text-xs font-semibold text-slate-100">{stat.value}</p>
          </div>
        ))}
      </section>

      {/* Route list */}
      {routes.length === 0 ? (
        <div className="rounded-lg border border-slate-700/70 bg-slate-900/40 p-8 text-center">
          <p className="text-sm text-slate-400">No routes yet. Create one to start routing traffic.</p>
          <Button onClick={openCreate} className="mt-4 px-3 py-1.5 text-xs">Create Route</Button>
        </div>
      ) : (
        <div className="space-y-2">
          {routes.map((route) => {
            const isExpanded = expandedId === route.id;
            return (
              <section
                key={route.id}
                className="overflow-hidden rounded-lg border border-slate-700/70 bg-slate-900/40"
              >
                {/* Row */}
                <div className="grid grid-cols-[minmax(0,1fr)_auto] items-center gap-3 px-3 py-2.5">
                  <button
                    type="button"
                    className="min-w-0 text-left"
                    onClick={() => setExpandedId(isExpanded ? null : route.id)}
                    aria-expanded={isExpanded}
                  >
                    <div className="flex flex-wrap items-center gap-2">
                      <span className="font-mono text-xs font-semibold text-slate-100">{route.name}</span>
                      <span className={`inline-flex rounded-sm px-1.5 py-0.5 text-[10px] font-medium ${STRATEGY_COLORS[route.policy.selection.strategy]}`}>
                        {route.policy.selection.strategy}
                      </span>
                      {route.disabled && (
                        <span className="inline-flex rounded-sm bg-slate-700/40 px-1.5 py-0.5 text-[10px] font-medium text-slate-400">
                          disabled
                        </span>
                      )}
                    </div>
                    {route.description && (
                      <p className="mt-0.5 truncate text-[11px] text-slate-500">{route.description}</p>
                    )}
                    <div className="mt-1 flex flex-wrap gap-1">
                      {route.targets.map((t, i) => (
                        <span key={i} className="inline-flex items-center gap-1 rounded-sm border border-slate-700/60 bg-slate-800/50 px-1.5 py-0.5 font-mono text-[10px] text-slate-300">
                          {t.providerRef}
                          <span className={`rounded-sm px-1 py-px text-[9px] ${MODE_COLORS[t.mode]}`}>{t.mode}</span>
                          {t.mode === "weighted" && t.weight != null && (
                            <span className="text-slate-500">{t.weight}%</span>
                          )}
                          {t.mode === "failover" && t.priority != null && (
                            <span className="text-slate-500">p{t.priority}</span>
                          )}
                        </span>
                      ))}
                      {route.targets.length === 0 && (
                        <span className="text-[10px] text-slate-600">no targets</span>
                      )}
                    </div>
                  </button>

                  <div className="flex shrink-0 items-center gap-1.5">
                    <button
                      type="button"
                      onClick={() => handleToggleDisabled(route.id)}
                      className={`rounded-full border px-2.5 py-0.5 text-[10px] font-semibold transition-colors ${
                        route.disabled
                          ? "border-slate-600/60 bg-slate-800/40 text-slate-400 hover:border-emerald-500/40 hover:text-emerald-300"
                          : "border-emerald-500/30 bg-emerald-500/10 text-emerald-300 hover:border-slate-600/60 hover:bg-slate-800/40 hover:text-slate-400"
                      }`}
                      title={route.disabled ? "Enable route" : "Disable route"}
                    >
                      {route.disabled ? "Enable" : "Active"}
                    </button>
                    <Button
                      variant="danger"
                      className="px-2 py-1 text-[10px]"
                      onClick={() => { setPendingDeleteId(route.id); setShowConfirm(true); }}
                    >
                      Delete
                    </Button>
                    <button
                      type="button"
                      onClick={() => setExpandedId(isExpanded ? null : route.id)}
                      className="rounded-md border border-slate-700/60 bg-slate-800/40 p-1 text-slate-400 transition-colors hover:bg-slate-700/60 hover:text-slate-200"
                      aria-label={isExpanded ? "Collapse" : "Expand"}
                    >
                      <svg
                        className={`h-3.5 w-3.5 transition-transform ${isExpanded ? "rotate-180" : ""}`}
                        fill="none" stroke="currentColor" strokeWidth={2} viewBox="0 0 24 24"
                      >
                        <path strokeLinecap="round" strokeLinejoin="round" d="M19 9l-7 7-7-7" />
                      </svg>
                    </button>
                  </div>
                </div>

                {/* Detail panel */}
                {isExpanded && (
                  <div className="border-t border-slate-700/60 bg-slate-950/30 px-3 py-3">
                    <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
                      {/* Auth & Policy */}
                      <div>
                        <p className="mb-1.5 text-[10px] font-semibold uppercase tracking-[0.12em] text-slate-500">Auth & Policy</p>
                        <div className="space-y-1 text-[11px] text-slate-300">
                          <div className="flex justify-between">
                            <span className="text-slate-500">Require local key</span>
                            <span className={route.policy.auth.requireLocalAPIKey ? "text-emerald-300" : "text-slate-400"}>
                              {route.policy.auth.requireLocalAPIKey ? "Yes" : "No"}
                            </span>
                          </div>
                          <div className="flex justify-between">
                            <span className="text-slate-500">Selection strategy</span>
                            <span className={`rounded-sm px-1 py-px text-[10px] ${STRATEGY_COLORS[route.policy.selection.strategy]}`}>
                              {route.policy.selection.strategy}
                            </span>
                          </div>
                          <div className="flex justify-between">
                            <span className="text-slate-500">Timeout</span>
                            <span>{route.policy.timeoutSeconds}s</span>
                          </div>
                          <div className="flex justify-between">
                            <span className="text-slate-500">Max retries</span>
                            <span>{route.policy.retry.maxAttempts}</span>
                          </div>
                        </div>
                      </div>

                      {/* Capabilities */}
                      <div>
                        <p className="mb-1.5 text-[10px] font-semibold uppercase tracking-[0.12em] text-slate-500">Capabilities</p>
                        <div className="space-y-1 text-[11px]">
                          {[
                            { label: "Streaming", value: route.policy.allowStreaming },
                            { label: "Tools / Function calls", value: route.policy.allowTools },
                          ].map(({ label, value }) => (
                            <div key={label} className="flex justify-between">
                              <span className="text-slate-500">{label}</span>
                              {value == null ? (
                                <span className="text-slate-600">—</span>
                              ) : (
                                <span className={value ? "text-emerald-300" : "text-slate-400"}>{value ? "Allowed" : "Blocked"}</span>
                              )}
                            </div>
                          ))}
                        </div>
                      </div>

                      {/* Targets detail */}
                      <div>
                        <p className="mb-1.5 text-[10px] font-semibold uppercase tracking-[0.12em] text-slate-500">
                          Targets ({route.targets.length})
                        </p>
                        {route.targets.length === 0 ? (
                          <p className="text-[11px] text-slate-600">No targets configured.</p>
                        ) : (
                          <div className="space-y-1">
                            {route.targets.map((t, i) => (
                              <div key={i} className="flex items-center justify-between rounded-sm border border-slate-700/50 bg-slate-900/40 px-2 py-1">
                                <span className="font-mono text-[11px] text-slate-200">{t.providerRef}</span>
                                <div className="flex items-center gap-1.5">
                                  <span className={`rounded-sm px-1.5 py-px text-[9px] font-medium ${MODE_COLORS[t.mode]}`}>{t.mode}</span>
                                  {t.mode === "weighted" && t.weight != null && (
                                    <span className="text-[10px] text-slate-400">w={t.weight}</span>
                                  )}
                                  {t.mode === "failover" && t.priority != null && (
                                    <span className="text-[10px] text-slate-400">p={t.priority}</span>
                                  )}
                                  {t.disabled && (
                                    <span className="text-[9px] text-slate-600">off</span>
                                  )}
                                </div>
                              </div>
                            ))}
                          </div>
                        )}
                      </div>
                    </div>

                    <div className="mt-3 flex items-center justify-between border-t border-slate-700/50 pt-2">
                      <span className="text-[10px] text-slate-600">
                        ID: <span className="font-mono text-slate-500">{route.id}</span>
                        {" · "}Created {new Date(route.createdAt).toLocaleDateString()}
                        {" · "}Updated {new Date(route.updatedAt).toLocaleDateString()}
                      </span>
                    </div>
                  </div>
                )}
              </section>
            );
          })}
        </div>
      )}

      {/* Create modal */}
      <Modal isOpen={isCreateOpen} onClose={() => setIsCreateOpen(false)}>
        <ModalHeader><ModalTitle>Create Route</ModalTitle></ModalHeader>
        <ModalContent>
          <div className="space-y-4">
            <div>
              <label className="mb-1.5 block text-sm font-medium text-slate-300">
                Route Name <span className="text-red-400">*</span>
              </label>
              <Input name="name" value={formName} onChange={setFormName} placeholder="e.g. default, ha-fallback" />
              <p className="mt-1 text-xs text-slate-500">Used as the route ID in Caddyfile and API requests.</p>
            </div>
            <div>
              <label className="mb-1.5 block text-sm font-medium text-slate-300">Description</label>
              <Input name="description" value={formDesc} onChange={setFormDesc} placeholder="Optional description" />
            </div>
            <div>
              <label className="mb-1.5 block text-sm font-medium text-slate-300">Initial Provider</label>
              <Input name="provider" value={formProvider} onChange={setFormProvider} placeholder="e.g. openai, anthropic, ollama" />
              <p className="mt-1 text-xs text-slate-500">Provider ref to add as the first target (optional).</p>
            </div>
            <div>
              <label className="mb-1.5 block text-sm font-medium text-slate-300">
                Selection Strategy
                <HelpTooltip content="auto: conditional → weighted; weighted: distribute by weight; failover: by priority; conditional: capability-based" />
              </label>
              <select
                value={formStrategy}
                onChange={(e) => setFormStrategy(e.target.value as SelectionStrategy)}
                className="w-full rounded-md border border-slate-700/70 bg-slate-900/60 px-3 py-2 text-sm text-slate-100 focus:border-blue-500/60 focus:outline-none"
              >
                <option value="auto">auto</option>
                <option value="weighted">weighted</option>
                <option value="failover">failover</option>
                <option value="conditional">conditional</option>
              </select>
            </div>
            <div>
              <label className="mb-1.5 block text-sm font-medium text-slate-300">Timeout (seconds)</label>
              <Input name="timeout" value={formTimeout} onChange={setFormTimeout} placeholder="120" />
            </div>
            <div className="flex items-center gap-2.5">
              <input
                id="require-key"
                type="checkbox"
                checked={formRequireKey}
                onChange={(e) => setFormRequireKey(e.target.checked)}
                className="h-3.5 w-3.5 rounded border-slate-600 bg-slate-800 accent-blue-500"
              />
              <label htmlFor="require-key" className="text-sm text-slate-300">
                Require local API key
                <HelpTooltip content="When enabled, callers must present a gateway local API key in the Authorization header." />
              </label>
            </div>
          </div>
        </ModalContent>
        <ModalFooter>
          <Button variant="ghost" onClick={() => setIsCreateOpen(false)}>Cancel</Button>
          <Button onClick={handleCreate}>Create Route</Button>
        </ModalFooter>
      </Modal>

      <ConfirmDialog
        isOpen={showConfirm}
        onClose={() => { setShowConfirm(false); setPendingDeleteId(null); }}
        onConfirm={handleDelete}
        title="Delete Route"
        message="Are you sure you want to delete this route? All associated targets and policy will be removed."
        confirmLabel="Delete"
        variant="danger"
      />
    </div>
  );
}
