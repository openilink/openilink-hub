import { Link, Outlet, useLocation, useNavigate } from "react-router-dom";
import { useEffect, useState } from "react";
import { LayoutDashboard, Bot, Cable, LogOut } from "lucide-react";
import { api } from "../lib/api";

const nav = [
  { label: "概览", href: "/", icon: LayoutDashboard },
  { label: "Bot", href: "/bots", icon: Bot },
  { label: "分发通道", href: "/sublevels", icon: Cable },
];

export function Layout() {
  const location = useLocation();
  const navigate = useNavigate();
  const [user, setUser] = useState<any>(null);

  useEffect(() => {
    api.me().then(setUser).catch(() => navigate("/login"));
  }, []);

  if (!user) return null;

  async function handleLogout() {
    await api.logout();
    navigate("/login");
  }

  return (
    <div className="min-h-screen flex">
      <aside className="w-56 border-r border-[var(--border)] flex flex-col shrink-0">
        <div className="p-4 border-b border-[var(--border)]">
          <h1 className="font-semibold text-sm">OpenILink Hub</h1>
        </div>
        <nav className="flex-1 p-2 space-y-1">
          {nav.map((item) => {
            const active = location.pathname === item.href;
            return (
              <Link
                key={item.href}
                to={item.href}
                className={`flex items-center gap-3 px-3 py-2 rounded-lg text-sm transition-colors ${
                  active
                    ? "bg-[var(--secondary)] text-[var(--foreground)]"
                    : "text-[var(--muted-foreground)] hover:bg-[var(--secondary)] hover:text-[var(--foreground)]"
                }`}
              >
                <item.icon className="w-4 h-4" />
                {item.label}
              </Link>
            );
          })}
        </nav>
        <div className="p-3 border-t border-[var(--border)] flex items-center justify-between">
          <span className="text-xs text-[var(--muted-foreground)] truncate">{user.username}</span>
          <button onClick={handleLogout} className="text-[var(--muted-foreground)] hover:text-[var(--foreground)] cursor-pointer">
            <LogOut className="w-4 h-4" />
          </button>
        </div>
      </aside>
      <main className="flex-1 p-6 overflow-auto">
        <Outlet />
      </main>
    </div>
  );
}
