import { Outlet, useNavigate, Link, useLocation } from "react-router-dom";
import { useEffect, useState } from "react";
import { LogOut, Settings, Github, Puzzle, Bot, LayoutDashboard } from "lucide-react";
import { api } from "../lib/api";

const navItems = [
  { path: "/", icon: Bot, label: "Bot 管理" },
  { path: "/webhook-plugins", icon: Puzzle, label: "插件市场" },
  { path: "/settings", icon: Settings, label: "设置" },
];

export function Layout() {
  const navigate = useNavigate();
  const location = useLocation();
  const [user, setUser] = useState<any>(null);

  useEffect(() => {
    api.me().then(setUser).catch(() => navigate("/login"));
  }, []);

  if (!user) return null;

  async function handleLogout() {
    await api.logout();
    navigate("/login");
  }

  // Determine active nav item (match prefix for sub-pages like /bot/:id)
  function isActive(path: string) {
    if (path === "/") return location.pathname === "/" || location.pathname.startsWith("/bot/");
    return location.pathname.startsWith(path);
  }

  return (
    <div className="min-h-screen flex">
      {/* Sidebar */}
      <aside className="w-52 border-r flex flex-col shrink-0">
        {/* Logo */}
        <div className="px-4 py-4 border-b">
          <Link to="/" className="flex items-center gap-2 hover:opacity-80">
            <LayoutDashboard className="w-5 h-5 text-primary" />
            <span className="font-semibold text-sm">OpenILink Hub</span>
          </Link>
        </div>

        {/* Nav */}
        <nav className="flex-1 px-2 py-3 space-y-0.5">
          {navItems.map((item) => {
            // Plugin marketplace is public, skip if in Layout
            if (item.path === "/webhook-plugins") return null;
            const active = isActive(item.path);
            return (
              <Link key={item.path} to={item.path}
                className={`flex items-center gap-2.5 px-3 py-2 rounded-lg text-sm transition-colors ${
                  active ? "bg-secondary text-foreground font-medium" : "text-muted-foreground hover:text-foreground hover:bg-secondary/50"
                }`}>
                <item.icon className="w-4 h-4" />
                {item.label}
              </Link>
            );
          })}
          <a href="/webhook-plugins" className="flex items-center gap-2.5 px-3 py-2 rounded-lg text-sm text-muted-foreground hover:text-foreground hover:bg-secondary/50 transition-colors">
            <Puzzle className="w-4 h-4" />
            插件市场
          </a>
        </nav>

        {/* Footer */}
        <div className="border-t px-3 py-3 space-y-2">
          <div className="flex items-center gap-2 px-1">
            <div className="w-7 h-7 rounded-full bg-secondary flex items-center justify-center text-xs font-medium">
              {user.username.charAt(0).toUpperCase()}
            </div>
            <div className="flex-1 min-w-0">
              <p className="text-xs font-medium truncate">{user.username}</p>
              <p className="text-[10px] text-muted-foreground">{user.role === "admin" ? "管理员" : "成员"}</p>
            </div>
          </div>
          <div className="flex items-center gap-1">
            <a href="https://github.com/openilink/openilink-hub" target="_blank" rel="noopener"
              className="flex-1 flex items-center justify-center gap-1 text-[10px] text-muted-foreground hover:text-foreground py-1 rounded hover:bg-secondary/50 transition-colors">
              <Github className="w-3 h-3" /> GitHub
            </a>
            <button onClick={handleLogout}
              className="flex-1 flex items-center justify-center gap-1 text-[10px] text-muted-foreground hover:text-foreground py-1 rounded hover:bg-secondary/50 transition-colors cursor-pointer">
              <LogOut className="w-3 h-3" /> 退出
            </button>
          </div>
        </div>
      </aside>

      {/* Main content */}
      <main className="flex-1 overflow-auto">
        <div className="max-w-4xl mx-auto p-6">
          <Outlet />
        </div>
      </main>
    </div>
  );
}
