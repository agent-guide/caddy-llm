const MOCK_MODELS = [
  { id: "gpt-4o", provider: "openai", type: "chat", contextWindow: 128000, status: "active" },
  { id: "gpt-4o-mini", provider: "openai", type: "chat", contextWindow: 128000, status: "active" },
  { id: "claude-opus-4-6", provider: "anthropic", type: "chat", contextWindow: 200000, status: "active" },
  { id: "claude-sonnet-4-6", provider: "anthropic", type: "chat", contextWindow: 200000, status: "active" },
  { id: "claude-haiku-4-5", provider: "anthropic", type: "chat", contextWindow: 200000, status: "active" },
  { id: "llama3.2", provider: "ollama", type: "chat", contextWindow: 8192, status: "inactive" },
];

export default function ModelsPage() {
  return (
    <div className="space-y-4">
      <section className="rounded-lg border border-slate-700/70 bg-slate-900/40 p-4">
        <h1 className="text-xl font-semibold tracking-tight text-slate-100">Models</h1>
        <p className="mt-1 text-sm text-slate-400">Available LLM models across all configured providers.</p>
      </section>

      <div className="grid grid-cols-2 gap-2 lg:grid-cols-4">
        {[
          { label: "Total Models", value: MOCK_MODELS.length },
          { label: "Active", value: MOCK_MODELS.filter((m) => m.status === "active").length },
          { label: "Providers", value: new Set(MOCK_MODELS.map((m) => m.provider)).size },
          { label: "Inactive", value: MOCK_MODELS.filter((m) => m.status === "inactive").length },
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
              {["Model ID", "Provider", "Type", "Context Window", "Status"].map((h) => (
                <th key={h} className="px-3 py-2 text-left text-[10px] font-semibold uppercase tracking-[0.08em] text-slate-400">{h}</th>
              ))}
            </tr>
          </thead>
          <tbody>
            {MOCK_MODELS.map((model) => (
              <tr key={model.id} className="border-b border-slate-700/60 last:border-b-0 hover:bg-slate-800/30 transition-colors">
                <td className="px-3 py-2 font-mono text-xs text-slate-100">{model.id}</td>
                <td className="px-3 py-2 text-xs text-slate-300 capitalize">{model.provider}</td>
                <td className="px-3 py-2 text-xs text-slate-400">{model.type}</td>
                <td className="px-3 py-2 text-xs text-slate-400">{model.contextWindow.toLocaleString()}</td>
                <td className="px-3 py-2">
                  <span className={`inline-flex items-center rounded-sm px-1.5 py-0.5 text-[10px] font-medium ${model.status === "active" ? "bg-emerald-500/15 text-emerald-300" : "bg-slate-700/40 text-slate-400"}`}>
                    {model.status}
                  </span>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </section>
    </div>
  );
}
