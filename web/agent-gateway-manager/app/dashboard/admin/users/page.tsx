"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Modal, ModalHeader, ModalTitle, ModalContent, ModalFooter } from "@/components/ui/modal";
import { useToast } from "@/components/ui/toast";
import { Breadcrumbs } from "@/components/ui/breadcrumbs";

interface User {
  id: string;
  username: string;
  isAdmin: boolean;
  createdAt: string;
  apiKeyCount: number;
}

const MOCK_USERS: User[] = [
  { id: "1", username: "admin", isAdmin: true, createdAt: "2026-01-01", apiKeyCount: 2 },
  { id: "2", username: "alice", isAdmin: false, createdAt: "2026-02-15", apiKeyCount: 1 },
  { id: "3", username: "bob", isAdmin: false, createdAt: "2026-03-01", apiKeyCount: 0 },
];

export default function AdminUsersPage() {
  const [users, setUsers] = useState<User[]>(MOCK_USERS);
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [isAdmin, setIsAdmin] = useState(false);
  const { showToast } = useToast();

  const handleCreateUser = () => {
    if (password !== confirmPassword) { showToast("Passwords do not match", "error"); return; }
    if (password.length < 8) { showToast("Password must be at least 8 characters", "error"); return; }
    if (!username.trim()) { showToast("Username is required", "error"); return; }
    setUsers((prev) => [...prev, {
      id: Date.now().toString(), username, isAdmin, createdAt: new Date().toISOString().split("T")[0], apiKeyCount: 0,
    }]);
    showToast("User created successfully", "success");
    setIsModalOpen(false);
    setUsername(""); setPassword(""); setConfirmPassword(""); setIsAdmin(false);
  };

  return (
    <div className="space-y-4">
      <Breadcrumbs items={[{ label: "Dashboard", href: "/dashboard" }, { label: "Admin" }, { label: "Users" }]} />
      <section className="rounded-lg border border-slate-700/70 bg-slate-900/40 p-4">
        <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <h1 className="text-xl font-semibold tracking-tight text-slate-100">User Management</h1>
            <p className="mt-1 text-xs text-slate-400">Manage dashboard users and roles.</p>
          </div>
          <Button onClick={() => setIsModalOpen(true)} className="px-2.5 py-1 text-xs">Create User</Button>
        </div>
      </section>

      <section className="overflow-x-auto rounded-lg border border-slate-700/70 bg-slate-900/40">
        <table className="min-w-[600px] w-full text-sm">
          <thead>
            <tr className="border-b border-slate-700/70 bg-slate-900/95 backdrop-blur-sm">
              <th className="px-3 py-2 text-left text-[10px] font-semibold uppercase tracking-[0.08em] text-slate-400">Username</th>
              <th className="px-3 py-2 text-left text-[10px] font-semibold uppercase tracking-[0.08em] text-slate-400">Role</th>
              <th className="px-3 py-2 text-left text-[10px] font-semibold uppercase tracking-[0.08em] text-slate-400">Created</th>
              <th className="px-3 py-2 text-left text-[10px] font-semibold uppercase tracking-[0.08em] text-slate-400">API Keys</th>
            </tr>
          </thead>
          <tbody>
            {users.map((user) => (
              <tr key={user.id} className="border-b border-slate-700/60 last:border-b-0 hover:bg-slate-800/30 transition-colors">
                <td className="px-3 py-2 text-xs font-medium text-slate-100">{user.username}</td>
                <td className="px-3 py-2">
                  <span className={`inline-flex items-center rounded-sm border px-2 py-0.5 text-xs font-medium ${user.isAdmin ? "border-blue-500/40 bg-blue-500/10 text-blue-200" : "border-slate-600/70 bg-slate-700/40 text-slate-300"}`}>
                    {user.isAdmin ? "Admin" : "User"}
                  </span>
                </td>
                <td className="px-3 py-2 text-xs text-slate-400">{new Date(user.createdAt).toLocaleDateString("en-US", { year: "numeric", month: "short", day: "numeric" })}</td>
                <td className="px-3 py-2 text-xs text-slate-300">{user.apiKeyCount}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </section>

      <Modal isOpen={isModalOpen} onClose={() => setIsModalOpen(false)}>
        <ModalHeader><ModalTitle>Create New User</ModalTitle></ModalHeader>
        <ModalContent>
          <div className="space-y-4">
            <div>
              <label className="mb-2 block text-sm font-medium text-slate-300">Username</label>
              <Input name="username" value={username} onChange={setUsername} placeholder="Enter username" autoComplete="username" />
            </div>
            <div>
              <label className="mb-2 block text-sm font-medium text-slate-300">Password</label>
              <Input type="password" name="password" value={password} onChange={setPassword} placeholder="Minimum 8 characters" autoComplete="new-password" />
            </div>
            <div>
              <label className="mb-2 block text-sm font-medium text-slate-300">Confirm Password</label>
              <Input type="password" name="confirmPassword" value={confirmPassword} onChange={setConfirmPassword} placeholder="Re-enter password" autoComplete="new-password" />
            </div>
            <label className="flex items-center gap-3 cursor-pointer">
              <input type="checkbox" checked={isAdmin} onChange={(e) => setIsAdmin(e.target.checked)}
                className="size-4 shrink-0 cursor-pointer rounded border-slate-600/70 bg-slate-900/40 text-blue-600 focus:ring-2 focus:ring-blue-500 focus:ring-offset-0" />
              <span className="text-sm font-medium text-slate-200">Grant admin privileges</span>
            </label>
          </div>
        </ModalContent>
        <ModalFooter>
          <Button variant="secondary" onClick={() => setIsModalOpen(false)}>Cancel</Button>
          <Button onClick={handleCreateUser}>Create User</Button>
        </ModalFooter>
      </Modal>
    </div>
  );
}
