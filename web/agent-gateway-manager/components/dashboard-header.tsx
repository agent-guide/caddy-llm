"use client";

interface DashboardHeaderProps {
  onUserClick: () => void;
  username: string;
  isAdmin: boolean;
}

export function DashboardHeader({ onUserClick, username, isAdmin }: DashboardHeaderProps) {
  const initial = username ? username.charAt(0).toUpperCase() : "?";
  return (
    <header className="w-full bg-slate-900/40 border-b border-slate-700/70 backdrop-blur-sm py-2.5 px-4 lg:px-6 rounded-lg mb-4 flex items-center justify-between">
      <div className="flex items-center gap-3 text-sm">
        <div className="flex items-center gap-2">
          <div className="w-2.5 h-2.5 rounded-full bg-emerald-500 animate-pulse-dot" />
          <span className="text-emerald-400 font-medium">All systems operational</span>
        </div>
      </div>
      <div className="flex items-center gap-2">
        <button
          onClick={onUserClick}
          aria-label="User settings"
          className="flex items-center gap-3 group transition-all"
        >
          <div className="hidden sm:flex flex-col items-end">
            <span className="text-sm font-medium text-slate-200 group-hover:text-white transition-colors">{username}</span>
            {isAdmin && <span className="text-[10px] font-bold uppercase tracking-wider text-blue-400">Admin</span>}
          </div>
          <div className="w-9 h-9 rounded-full bg-slate-800/60 border border-slate-600/50 flex items-center justify-center text-sm font-medium text-slate-200 group-hover:border-blue-400/50 group-hover:shadow-[0_0_10px_rgba(96,165,250,0.2)] transition-all">
            {initial}
          </div>
        </button>
      </div>
    </header>
  );
}
