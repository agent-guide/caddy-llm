"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Modal, ModalHeader, ModalTitle, ModalContent, ModalFooter } from "@/components/ui/modal";
import { useToast } from "@/components/ui/toast";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";

interface Provider {
  id: string;
  name: string;
  type: string;
  baseUrl: string;
  modelCount: number;
  status: "active" | "inactive";
}

const MOCK_PROVIDERS: Provider[] = [
  { id: "1", name: "OpenAI", type: "openai", baseUrl: "https://api.openai.com/v1", modelCount: 8, status: "active" },
  { id: "2", name: "Anthropic", type: "anthropic", baseUrl: "https://api.anthropic.com", modelCount: 4, status: "active" },
  { id: "3", name: "Ollama Local", type: "ollama", baseUrl: "http://localhost:11434", modelCount: 3, status: "inactive" },
];

export default function ProvidersPage() {
  const [providers, setProviders] = useState<Provider[]>(MOCK_PROVIDERS);
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [showConfirm, setShowConfirm] = useState(false);
  const [pendingDeleteId, setPendingDeleteId] = useState<string | null>(null);
  const [name, setName] = useState("");
  const [baseUrl, setBaseUrl] = useState("");
  const [apiKey, setApiKey] = useState("");
  const { showToast } = useToast();

  const activeCount = providers.filter((p) => p.status === "active").length;

  const handleCreate = () => {
    if (!name.trim()) { showToast("Provider name is required", "error"); return; }
    const newProvider: Provider = {
      id: Date.now().toString(), name, type: "custom", baseUrl, modelCount: 0, status: "active",
    };
    setProviders((prev) => [...prev, newProvider]);
    showToast("Provider added", "success");
    setIsModalOpen(false);
    setName(""); setBaseUrl(""); setApiKey("");
  };

  const handleDelete = () => {
    if (!pendingDeleteId) return;
    setProviders((prev) => prev.filter((p) => p.id !== pendingDeleteId));
    showToast("Provider removed", "success");
  };

  return (
    <div className="space-y-6">
      <section className="rounded-lg border border-slate-700/70 bg-slate-900/40 p-4">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-xl font-semibold tracking-tight text-slate-100">AI Providers</h1>
            <p className="mt-1 text-sm text-slate-400">Manage LLM provider backends and API keys.</p>
          </div>
          <Button onClick={() => setIsModalOpen(true)} className="px-2.5 py-1 text-xs">Add Provider</Button>
        </div>
      </section>

      <section className="grid grid-cols-2 gap-2 lg:grid-cols-4">
        {[
          { label: "Total Providers", value: providers.length },
          { label: "Active", value: activeCount },
          { label: "Total Models", value: providers.reduce((s, p) => s + p.modelCount, 0) },
          { label: "Inactive", value: providers.length - activeCount },
        ].map((stat) => (
          <div key={stat.label} className="rounded-lg border border-slate-700/70 bg-slate-900/40 px-2.5 py-2">
            <p className="text-[10px] font-semibold uppercase tracking-[0.12em] text-slate-500">{stat.label}</p>
            <p className="mt-0.5 text-xs font-semibold text-slate-100">{stat.value}</p>
          </div>
        ))}
      </section>

      <section className="overflow-x-auto rounded-lg border border-slate-700/70 bg-slate-900/40">
        <div className="sticky top-0 z-10 grid grid-cols-[minmax(0,1fr)_120px_200px_80px_100px] border-b border-slate-700/70 bg-slate-900/95 backdrop-blur-sm px-3 py-2 text-[10px] font-semibold uppercase tracking-[0.08em] text-slate-400">
          <span>Name</span><span>Type</span><span>Base URL</span><span>Models</span><span>Actions</span>
        </div>
        {providers.map((provider) => (
          <div key={provider.id} className="grid grid-cols-[minmax(0,1fr)_120px_200px_80px_100px] items-center border-b border-slate-700/60 px-3 py-2 last:border-b-0">
            <div className="min-w-0">
              <p className="truncate text-xs font-medium text-slate-100">{provider.name}</p>
              <span className={`inline-flex items-center rounded-sm px-1.5 py-0.5 text-[10px] font-medium ${provider.status === "active" ? "bg-emerald-500/15 text-emerald-300" : "bg-slate-700/40 text-slate-400"}`}>
                {provider.status}
              </span>
            </div>
            <span className="text-xs text-slate-400 font-mono">{provider.type}</span>
            <span className="truncate text-xs text-slate-400 font-mono">{provider.baseUrl}</span>
            <span className="text-xs text-slate-300">{provider.modelCount}</span>
            <div className="flex gap-1">
              <Button variant="danger" onClick={() => { setPendingDeleteId(provider.id); setShowConfirm(true); }} className="px-2 py-1 text-xs">Remove</Button>
            </div>
          </div>
        ))}
      </section>

      <Modal isOpen={isModalOpen} onClose={() => setIsModalOpen(false)}>
        <ModalHeader><ModalTitle>Add Provider</ModalTitle></ModalHeader>
        <ModalContent>
          <div className="space-y-4">
            <div>
              <label className="mb-2 block text-sm font-medium text-slate-300">Name</label>
              <Input name="name" value={name} onChange={setName} placeholder="e.g. My OpenAI" />
            </div>
            <div>
              <label className="mb-2 block text-sm font-medium text-slate-300">Base URL</label>
              <Input name="baseUrl" value={baseUrl} onChange={setBaseUrl} placeholder="https://api.openai.com/v1" />
            </div>
            <div>
              <label className="mb-2 block text-sm font-medium text-slate-300">API Key</label>
              <Input type="password" name="apiKey" value={apiKey} onChange={setApiKey} placeholder="sk-..." />
            </div>
          </div>
        </ModalContent>
        <ModalFooter>
          <Button variant="ghost" onClick={() => setIsModalOpen(false)}>Cancel</Button>
          <Button onClick={handleCreate}>Add Provider</Button>
        </ModalFooter>
      </Modal>

      <ConfirmDialog isOpen={showConfirm} onClose={() => { setShowConfirm(false); setPendingDeleteId(null); }}
        onConfirm={handleDelete} title="Remove Provider" message="Are you sure you want to remove this provider?" confirmLabel="Remove" variant="danger" />
    </div>
  );
}
