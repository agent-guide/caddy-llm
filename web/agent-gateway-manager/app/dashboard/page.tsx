import Link from "next/link";

const statusCards = [
  { label: "Service", value: "Online", tone: "text-emerald-400", icon: "●", iconTone: "text-emerald-300" },
  { label: "Providers", value: "3 configured", tone: "text-slate-100", icon: "◆", iconTone: "text-blue-300" },
  { label: "API Keys", value: "2 active", tone: "text-slate-100", icon: "♟", iconTone: "text-amber-300" },
  { label: "Models", value: "12 available", tone: "text-slate-100", icon: "◈", iconTone: "text-cyan-300" },
] as const;

export default function DashboardPage() {
  return (
    <div className="space-y-4">
      <section className="rounded-lg border border-slate-700/70 bg-slate-900/40 p-4">
        <div className="flex flex-col gap-3 lg:flex-row lg:items-start lg:justify-between">
          <div>
            <h1 className="text-xl font-semibold tracking-tight text-slate-100">Overview</h1>
            <p className="mt-1 text-sm text-slate-400">Monitor and manage your agent gateway from one place.</p>
          </div>
          <div className="flex flex-wrap gap-2">
            <Link href="/dashboard/providers" className="rounded-md border border-slate-600/80 bg-slate-800/70 px-3 py-1.5 text-xs font-semibold uppercase tracking-[0.1em] text-slate-200 transition-colors hover:bg-slate-700/80">
              Providers
            </Link>
            <Link href="/dashboard/api-keys" className="rounded-md border border-slate-600/80 bg-slate-800/70 px-3 py-1.5 text-xs font-semibold uppercase tracking-[0.1em] text-slate-200 transition-colors hover:bg-slate-700/80">
              API Keys
            </Link>
            <Link href="/dashboard/settings" className="rounded-md border border-slate-600/80 bg-slate-800/70 px-3 py-1.5 text-xs font-semibold uppercase tracking-[0.1em] text-slate-200 transition-colors hover:bg-slate-700/80">
              Settings
            </Link>
          </div>
        </div>
      </section>

      <section className="grid grid-cols-[repeat(auto-fit,minmax(150px,1fr))] gap-2">
        {statusCards.map((card) => (
          <div key={card.label} className="glass-card rounded-md border border-slate-700/70 px-2.5 py-2 transition-colors hover:border-slate-600">
            <div className="flex items-center justify-between">
              <div className="text-[11px] font-semibold uppercase tracking-[0.12em] text-slate-500">{card.label}</div>
              <span className={`text-xs ${card.iconTone}`} aria-hidden="true">{card.icon}</span>
            </div>
            <div className={`mt-0.5 text-xs font-semibold ${card.tone}`}>{card.value}</div>
          </div>
        ))}
      </section>

      <div className="grid gap-4 lg:grid-cols-2">
        <section className="rounded-lg border border-slate-700/70 bg-slate-900/40 p-4">
          <h2 className="mb-3 text-sm font-semibold text-slate-100">Quick Start</h2>
          <div className="space-y-2">
            {[
              { step: "1", text: "Configure a provider (OpenAI, Anthropic, etc.)", href: "/dashboard/providers" },
              { step: "2", text: "Create an API key for authentication", href: "/dashboard/api-keys" },
              { step: "3", text: "Configure your Caddyfile to use the LLM module", href: "/dashboard/settings" },
            ].map((item) => (
              <Link key={item.step} href={item.href} className="flex items-start gap-3 rounded-md border border-slate-700/50 bg-slate-900/30 p-3 hover:border-slate-600/70 hover:bg-slate-800/40 transition-colors">
                <span className="flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-blue-500/20 text-[10px] font-bold text-blue-300">{item.step}</span>
                <span className="text-xs text-slate-300">{item.text}</span>
              </Link>
            ))}
          </div>
        </section>

        <section className="rounded-lg border border-slate-700/70 bg-slate-900/40 p-4">
          <h2 className="mb-3 text-sm font-semibold text-slate-100">Integration Snippet</h2>
          <p className="mb-3 text-xs text-slate-400">Configure Claude Code to use the agent gateway as a proxy:</p>
          <div className="rounded-md border border-slate-700/70 bg-slate-950/60 p-3 font-mono text-xs text-slate-300">
            <div className="text-slate-500"># Environment variables</div>
            <div>export ANTHROPIC_BASE_URL=http://localhost:8080</div>
            <div>export ANTHROPIC_AUTH_TOKEN=your-api-key</div>
          </div>
        </section>
      </div>

      <section className="rounded-lg border border-slate-700/70 bg-slate-900/40 p-4">
        <h2 className="mb-3 text-sm font-semibold text-slate-100">System Modules</h2>
        <div className="grid grid-cols-2 gap-2 sm:grid-cols-3 lg:grid-cols-6">
          {[
            { label: "Providers", href: "/dashboard/providers", desc: "LLM backends" },
            { label: "Models", href: "/dashboard/models", desc: "Model catalog" },
            { label: "Agents", href: "/dashboard/agents", desc: "Agent mode" },
            { label: "Memory", href: "/dashboard/memory", desc: "Memory store" },
            { label: "API Keys", href: "/dashboard/api-keys", desc: "Auth tokens" },
            { label: "Monitoring", href: "/dashboard/monitoring", desc: "System health" },
          ].map((mod) => (
            <Link key={mod.label} href={mod.href} className="rounded-md border border-slate-700/60 bg-slate-900/30 p-3 hover:border-blue-500/40 hover:bg-blue-500/5 transition-colors">
              <div className="text-xs font-semibold text-slate-200">{mod.label}</div>
              <div className="mt-0.5 text-[10px] text-slate-500">{mod.desc}</div>
            </Link>
          ))}
        </div>
      </section>
    </div>
  );
}
