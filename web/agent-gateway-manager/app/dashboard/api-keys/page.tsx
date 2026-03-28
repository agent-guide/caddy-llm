"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Modal, ModalHeader, ModalTitle, ModalContent, ModalFooter } from "@/components/ui/modal";
import { useToast } from "@/components/ui/toast";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import { HelpTooltip } from "@/components/ui/tooltip";

interface ApiKey {
  id: string;
  name: string;
  keyPreview: string;
  createdAt: string;
  lastUsedAt: string | null;
}

const MOCK_KEYS: ApiKey[] = [
  { id: "1", name: "Development", keyPreview: "sk-***...abc1", createdAt: "2026-01-15", lastUsedAt: "2026-03-27" },
  { id: "2", name: "Production", keyPreview: "sk-***...xyz9", createdAt: "2026-02-01", lastUsedAt: null },
];

export default function ApiKeysPage() {
  const [apiKeys, setApiKeys] = useState<ApiKey[]>(MOCK_KEYS);
  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false);
  const [keyNameInput, setKeyNameInput] = useState("");
  const [showNewKey, setShowNewKey] = useState(false);
  const [newKeyValue, setNewKeyValue] = useState<string | null>(null);
  const [showConfirm, setShowConfirm] = useState(false);
  const [pendingDeleteId, setPendingDeleteId] = useState<string | null>(null);
  const { showToast } = useToast();

  const handleCreateKey = () => {
    const newKey = `sk-agent-${Math.random().toString(36).substring(2, 18)}`;
    const newApiKey: ApiKey = {
      id: Date.now().toString(),
      name: keyNameInput.trim() || "Default",
      keyPreview: `${newKey.substring(0, 8)}...${newKey.substring(newKey.length - 4)}`,
      createdAt: new Date().toISOString().split("T")[0],
      lastUsedAt: null,
    };
    setApiKeys((prev) => [...prev, newApiKey]);
    setNewKeyValue(newKey);
    setIsCreateModalOpen(false);
    setShowNewKey(true);
    showToast("API key created", "success");
  };

  const handleDelete = () => {
    if (!pendingDeleteId) return;
    setApiKeys((prev) => prev.filter((k) => k.id !== pendingDeleteId));
    showToast("API key deleted", "success");
  };

  return (
    <div className="space-y-4">
      <section className="rounded-lg border border-slate-700/70 bg-slate-900/40 p-4">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-xl font-semibold tracking-tight text-slate-100">API Keys</h1>
            <p className="mt-1 text-xs text-slate-400">
              Manage gateway access keys for clients and integrations.
              <HelpTooltip content="API keys authenticate requests to the agent gateway" />
            </p>
          </div>
          <Button onClick={() => { setKeyNameInput(""); setIsCreateModalOpen(true); }} className="px-2.5 py-1 text-xs">
            Create Key
          </Button>
        </div>
      </section>

      {apiKeys.length === 0 ? (
        <div className="rounded-lg border border-slate-700/70 bg-slate-900/40 p-8 text-center">
          <p className="text-sm text-slate-400">No API keys yet. Create one to get started.</p>
          <Button onClick={() => setIsCreateModalOpen(true)} className="mt-4 px-3 py-1.5 text-xs">Create API Key</Button>
        </div>
      ) : (
        <div className="overflow-x-auto">
          <section className="min-w-[600px] overflow-hidden rounded-lg border border-slate-700/70 bg-slate-900/40">
            <div className="sticky top-0 z-10 grid grid-cols-[minmax(0,1fr)_180px_160px_110px] border-b border-slate-700/70 bg-slate-900/95 backdrop-blur-sm px-3 py-2 text-[10px] font-semibold uppercase tracking-[0.08em] text-slate-400">
              <span>Name</span><span>Created</span><span>Last Used</span><span>Actions</span>
            </div>
            {apiKeys.map((apiKey) => (
              <div key={apiKey.id} className="grid grid-cols-[minmax(0,1fr)_180px_160px_110px] items-center border-b border-slate-700/60 px-3 py-2 last:border-b-0">
                <div className="min-w-0">
                  <p className="truncate text-xs font-medium text-slate-100">{apiKey.name}</p>
                  <p className="mt-0.5 truncate font-mono text-xs text-slate-400">{apiKey.keyPreview}</p>
                </div>
                <span className="text-xs text-slate-400">{new Date(apiKey.createdAt).toLocaleDateString()}</span>
                <span className="text-xs text-slate-400">{apiKey.lastUsedAt ? new Date(apiKey.lastUsedAt).toLocaleDateString() : "Never"}</span>
                <div className="flex justify-end">
                  <Button variant="danger" onClick={() => { setPendingDeleteId(apiKey.id); setShowConfirm(true); }} className="px-2.5 py-1 text-xs">Delete</Button>
                </div>
              </div>
            ))}
          </section>
        </div>
      )}

      <Modal isOpen={isCreateModalOpen} onClose={() => setIsCreateModalOpen(false)}>
        <ModalHeader><ModalTitle>Create API Key</ModalTitle></ModalHeader>
        <ModalContent>
          <label className="mb-2 block text-sm font-semibold text-slate-300">Key Name</label>
          <Input name="key-name" value={keyNameInput} onChange={setKeyNameInput} placeholder="e.g. Development, Production" />
          <p className="mt-1.5 text-xs text-slate-500">Give your key a descriptive name</p>
        </ModalContent>
        <ModalFooter>
          <Button variant="ghost" onClick={() => setIsCreateModalOpen(false)}>Cancel</Button>
          <Button onClick={handleCreateKey}>Create Key</Button>
        </ModalFooter>
      </Modal>

      <Modal isOpen={showNewKey && newKeyValue !== null} onClose={() => { setShowNewKey(false); setNewKeyValue(null); }}>
        <ModalHeader><ModalTitle>New API Key</ModalTitle></ModalHeader>
        <ModalContent>
          <div className="rounded-sm border border-slate-700/70 bg-slate-900/40 p-4 text-sm">
            <div className="mb-2 font-medium text-slate-100">Copy this key now</div>
            <div className="break-all rounded-sm border border-slate-700/70 bg-slate-900/40 p-3 font-mono text-xs text-slate-200">{newKeyValue}</div>
          </div>
          <div className="mt-3 rounded-sm border border-amber-500/40 bg-amber-500/10 p-3 text-sm">
            <span className="text-amber-200">This key will only be shown once. Store it securely.</span>
          </div>
        </ModalContent>
        <ModalFooter>
          <Button onClick={() => { setShowNewKey(false); setNewKeyValue(null); }}>I have saved it</Button>
        </ModalFooter>
      </Modal>

      <ConfirmDialog isOpen={showConfirm} onClose={() => { setShowConfirm(false); setPendingDeleteId(null); }}
        onConfirm={handleDelete} title="Delete API Key" message="Are you sure you want to delete this API key?" confirmLabel="Delete" variant="danger" />
    </div>
  );
}
