"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { useToast } from "@/components/ui/toast";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";

export default function SettingsPage() {
  const [caddyAddr, setCaddyAddr] = useState(":2019");
  const [logLevel, setLogLevel] = useState("info");
  const [showConfirmRevoke, setShowConfirmRevoke] = useState(false);
  const { showToast } = useToast();

  return (
    <div className="space-y-6">
      <section className="rounded-lg border border-slate-700/70 bg-slate-900/40 p-4">
        <h1 className="text-xl font-semibold tracking-tight text-slate-100">Settings</h1>
        <p className="mt-1 text-sm text-slate-400">Manage gateway configuration and system settings.</p>
      </section>

      <div className="flex flex-col lg:flex-row gap-6">
        <section className="rounded-lg border border-slate-700/70 bg-slate-900/40 p-6 flex-1 space-y-4">
          <div>
            <h2 className="text-sm font-semibold text-slate-100">Gateway Configuration</h2>
            <p className="text-xs text-slate-400 mt-1">Core agent gateway settings.</p>
          </div>
          <div className="space-y-3">
            <div>
              <label className="mb-1.5 block text-xs font-medium text-slate-300">Admin Address</label>
              <Input name="caddy-addr" value={caddyAddr} onChange={setCaddyAddr} placeholder=":2019" />
              <p className="mt-1 text-xs text-slate-500">Caddy admin API listen address</p>
            </div>
            <div>
              <label className="mb-1.5 block text-xs font-medium text-slate-300">Log Level</label>
              <select
                value={logLevel}
                onChange={(e) => setLogLevel(e.target.value)}
                className="w-full rounded-md glass-input text-white text-sm px-3 py-2 focus:outline-none focus:border-blue-400/50"
              >
                {["debug", "info", "warn", "error"].map((l) => (
                  <option key={l} value={l} className="bg-slate-900">{l}</option>
                ))}
              </select>
            </div>
          </div>
          <Button onClick={() => showToast("Settings saved", "success")}>Save Settings</Button>
        </section>

        <section className="rounded-lg border border-slate-700/70 bg-slate-900/40 p-6 flex-1 space-y-4 lg:self-start">
          <div>
            <h2 className="text-sm font-semibold text-slate-100">Security</h2>
            <p className="text-xs text-slate-400 mt-1">Session and access control.</p>
          </div>
          <div className="rounded-md border border-slate-700/60 bg-slate-900/30 p-4 space-y-3">
            <div>
              <p className="text-sm font-medium text-slate-200">Force Logout All Users</p>
              <p className="text-xs text-slate-400 mt-0.5">Revoke all active sessions immediately.</p>
            </div>
            <Button variant="danger" onClick={() => setShowConfirmRevoke(true)}>Revoke All Sessions</Button>
          </div>
          <div className="rounded-md border border-slate-700/60 bg-slate-900/30 p-4 space-y-3">
            <div>
              <p className="text-sm font-medium text-slate-200">Version Info</p>
              <p className="text-xs text-slate-400 mt-0.5">Agent Gateway Manager — <span className="text-slate-300 font-mono">v0.1.0</span></p>
            </div>
          </div>
        </section>
      </div>

      <ConfirmDialog
        isOpen={showConfirmRevoke}
        onClose={() => setShowConfirmRevoke(false)}
        onConfirm={() => showToast("All sessions revoked", "success")}
        title="Force Logout All Users"
        message="Force logout all users from all devices? This action cannot be undone."
        confirmLabel="Force Logout"
        variant="danger"
      />
    </div>
  );
}
