"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Modal, ModalHeader, ModalTitle, ModalContent, ModalFooter } from "@/components/ui/modal";
import { Input } from "@/components/ui/input";
import { useToast } from "@/components/ui/toast";

interface Agent {
  id: string;
  name: string;
  model: string;
  systemPrompt: string;
  status: "active" | "inactive";
}

const MOCK_AGENTS: Agent[] = [
  { id: "1", name: "Code Assistant", model: "claude-sonnet-4-6", systemPrompt: "You are an expert software engineer...", status: "active" },
  { id: "2", name: "Data Analyst", model: "gpt-4o", systemPrompt: "You are a data analysis expert...", status: "active" },
];

export default function AgentsPage() {
  const [agents, setAgents] = useState<Agent[]>(MOCK_AGENTS);
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [name, setName] = useState("");
  const [model, setModel] = useState("claude-sonnet-4-6");
  const { showToast } = useToast();

  const handleCreate = () => {
    if (!name.trim()) { showToast("Agent name is required", "error"); return; }
    setAgents((prev) => [...prev, { id: Date.now().toString(), name, model, systemPrompt: "", status: "active" }]);
    showToast("Agent created", "success");
    setIsModalOpen(false);
    setName(""); setModel("claude-sonnet-4-6");
  };

  return (
    <div className="space-y-4">
      <section className="rounded-lg border border-slate-700/70 bg-slate-900/40 p-4">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-xl font-semibold tracking-tight text-slate-100">Agents</h1>
            <p className="mt-1 text-sm text-slate-400">Configure and manage AI agent instances.</p>
          </div>
          <Button onClick={() => setIsModalOpen(true)} className="px-2.5 py-1 text-xs">Create Agent</Button>
        </div>
      </section>

      <div className="grid gap-3 sm:grid-cols-2">
        {agents.map((agent) => (
          <div key={agent.id} className="rounded-lg border border-slate-700/70 bg-slate-900/40 p-4 space-y-2">
            <div className="flex items-center justify-between">
              <h3 className="text-sm font-semibold text-slate-100">{agent.name}</h3>
              <span className={`inline-flex items-center rounded-sm px-1.5 py-0.5 text-[10px] font-medium ${agent.status === "active" ? "bg-emerald-500/15 text-emerald-300" : "bg-slate-700/40 text-slate-400"}`}>
                {agent.status}
              </span>
            </div>
            <p className="text-xs text-slate-400 font-mono">Model: {agent.model}</p>
            {agent.systemPrompt && (
              <p className="text-xs text-slate-500 truncate">{agent.systemPrompt}</p>
            )}
            <div className="flex gap-2 pt-1">
              <Button variant="secondary" className="px-2.5 py-1 text-xs">Edit</Button>
              <Button variant="danger" onClick={() => { setAgents((prev) => prev.filter((a) => a.id !== agent.id)); showToast("Agent removed", "success"); }} className="px-2.5 py-1 text-xs">Remove</Button>
            </div>
          </div>
        ))}
      </div>

      <Modal isOpen={isModalOpen} onClose={() => setIsModalOpen(false)}>
        <ModalHeader><ModalTitle>Create Agent</ModalTitle></ModalHeader>
        <ModalContent>
          <div className="space-y-4">
            <div>
              <label className="mb-2 block text-sm font-medium text-slate-300">Agent Name</label>
              <Input name="name" value={name} onChange={setName} placeholder="e.g. Code Assistant" />
            </div>
            <div>
              <label className="mb-2 block text-sm font-medium text-slate-300">Model</label>
              <Input name="model" value={model} onChange={setModel} placeholder="claude-sonnet-4-6" />
            </div>
          </div>
        </ModalContent>
        <ModalFooter>
          <Button variant="ghost" onClick={() => setIsModalOpen(false)}>Cancel</Button>
          <Button onClick={handleCreate}>Create Agent</Button>
        </ModalFooter>
      </Modal>
    </div>
  );
}
