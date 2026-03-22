import { Outlet, useNavigate, Link, useLocation } from "react-router-dom";
import { useEffect, useState } from "react";
import { LogOut, Github, Puzzle, Bot, LayoutDashboard, User, Shield, Bug, Store, FolderOpen, ShieldCheck, BarChart3, Users, Settings, Blocks, Sun, Moon } from "lucide-react";
import { api } from "../lib/api";
import { useTheme } from "../lib/theme";

export function Layout() {
  const navigate = useNavigate();
  const location = useLocation();
  const [user, setUser] = useState<any>(null);
  const { resolvedTheme, setTheme } = useTheme();

  useEffect(() => {
    api.me().then(setUser).catch(() => navigate("/login", { replace: true }));
  }, []);

  if (!user) return null;

  const isAdmin = user.role === "admin" || user.role === "superadmin";

  async function handleLogout() {
    await api.logout();
    navigate("/login", { replace: true });
  }

  function isActive(path: string) {
    if (path === "/dashboard") return location.pathname === "/dashboard" || location.pathname.startsWith("/dashboard/bot/");
    if (path === "/dashboard/apps") return location.pathname.startsWith("/dashboard/apps");
    return location.pathname === path;
  }

  function navLink(path: string, label: string, icon: any, indent = false) {
    const active = isActive(path);
    const Icon = icon;
    return (
      <Link key={path} to={path}
        className={`flex items-center gap-2 rounded-lg text-sm transition-colors ${indent ? "pl-9 pr-3 py-1.5" : "px-3 py-2"} ${
          active ? "bg-secondary text-foreground font-medium" : "text-muted-foreground hover:text-foreground hover:bg-secondary/50"
        }`}>
        {!indent && <Icon className="w-4 h-4" />}
        {label}
      </Link>
    );
  }

  function sectionLabel(label: string, icon: any) {
    const Icon = icon;
    return (
      <div className="flex items-center gap-2 px-3 py-2 text-xs font-medium text-muted-foreground uppercase tracking-wide">
        <Icon className="w-3.5 h-3.5" />
        {label}
      </div>
    );
  }

  return (
    <div className="h-screen flex">
      <aside className="w-52 border-r flex flex-col shrink-0 h-screen sticky top-0">
        {/* Logo */}
        <div className="px-5 py-5 border-b shrink-0">
          <Link to="/dashboard" className="flex items-center gap-2 hover:opacity-80">
            <LayoutDashboard className="w-5 h-5 text-primary" />
            <span className="font-semibold text-base tracking-tight">OpenILink Hub</span>
          </Link>
        </div>

        {/* Nav */}
        <nav className="flex-1 px-2 py-3 space-y-0.5 overflow-y-auto">
          {navLink("/dashboard", "Bot 管理", Bot)}
          {navLink("/dashboard/apps", "App 管理", Blocks)}

          {sectionLabel("Webhook 插件", Puzzle)}
          {navLink("/dashboard/webhook-plugins", "市场", Store, true)}
          {navLink("/dashboard/webhook-plugins/my", "我的插件", FolderOpen, true)}
          {navLink("/dashboard/webhook-plugins/debug", "调试器", Bug, true)}
          {isAdmin && navLink("/dashboard/webhook-plugins/review", "审核", ShieldCheck, true)}
        </nav>

        {/* Bottom nav */}
        <div className="border-t px-2 py-2 space-y-0.5 shrink-0">
          {navLink("/dashboard/settings", "账号设置", User)}
          {isAdmin && sectionLabel("系统管理", Shield)}
          {isAdmin && navLink("/dashboard/admin", "概览", BarChart3, true)}
          {isAdmin && navLink("/dashboard/admin/users", "用户管理", Users, true)}
          {isAdmin && navLink("/dashboard/admin/config", "系统配置", Settings, true)}
        </div>

        {/* User */}
        <div className="border-t px-3 py-3 space-y-2 shrink-0">
          <div className="flex items-center gap-2 px-1">
            <div className="w-7 h-7 rounded-full bg-secondary flex items-center justify-center text-xs font-medium">
              {user.username.charAt(0).toUpperCase()}
            </div>
            <div className="flex-1 min-w-0">
              <p className="text-xs font-medium truncate">{user.username}</p>
              <p className="text-[10px] text-muted-foreground">{user.role === "superadmin" ? "超级管理员" : user.role === "admin" ? "管理员" : "成员"}</p>
            </div>
          </div>
          <div className="flex items-center gap-1">
            <a href="https://github.com/openilink/openilink-hub" target="_blank" rel="noopener"
              className="flex-1 flex items-center justify-center gap-1 text-xs text-muted-foreground hover:text-foreground py-1.5 rounded-lg hover:bg-secondary/50 transition-colors">
              <Github className="w-3 h-3" /> GitHub
            </a>
            <button
              onClick={() => setTheme(resolvedTheme === "dark" ? "light" : "dark")}
              className="flex items-center justify-center text-muted-foreground hover:text-foreground py-1 px-2 rounded hover:bg-secondary/50 transition-colors cursor-pointer"
              aria-label="切换主题"
            >
              {resolvedTheme === "dark" ? <Sun className="w-3 h-3" /> : <Moon className="w-3 h-3" />}
            </button>
            <button onClick={handleLogout}
              className="flex-1 flex items-center justify-center gap-1 text-xs text-muted-foreground hover:text-foreground py-1.5 rounded-lg hover:bg-secondary/50 transition-colors cursor-pointer">
              <LogOut className="w-3 h-3" /> 退出
            </button>
          </div>
        </div>
      </aside>

      <main className="flex-1 overflow-auto h-screen">
        <div className="mx-auto max-w-6xl px-6 py-8 sm:px-8 sm:py-10 lg:px-10">
          <Outlet />
        </div>
      </main>
    </div>
  );
}
