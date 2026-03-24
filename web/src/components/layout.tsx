import { Outlet, useNavigate, Link, useLocation } from "react-router-dom";
import { useEffect, useState } from "react";
import React from "react";
import {
  LogOut,
  Bot,
  LayoutDashboard,
  User,
  Bug,
  Store,
  FolderOpen,
  ShieldCheck,
  BarChart3,
  Users,
  Settings,
  Blocks,
  Sun,
  Moon,
  ChevronsUpDown,
} from "lucide-react";
import { api } from "../lib/api";
import { useTheme } from "../lib/theme";
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarInset,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarProvider,
  SidebarRail,
  SidebarTrigger,
} from "@/components/ui/sidebar";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip";

function HeaderIconButton({
  tooltip,
  onClick,
  asChild,
  children,
}: {
  tooltip: string;
  onClick?: () => void;
  asChild?: boolean;
  children: React.ReactNode;
}) {
  return (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger asChild>
          <Button
            variant="ghost"
            size="icon-sm"
            asChild={asChild}
            onClick={onClick}
            className="relative h-7 w-7 rounded-md text-foreground/80 transition-all duration-150 hover:text-foreground hover:bg-accent focus-visible:ring-1 focus-visible:ring-ring [&_svg]:size-[15px] [&_svg]:transition-transform [&_svg]:duration-150 hover:[&_svg]:scale-110"
          >
            {children}
          </Button>
        </TooltipTrigger>
        <TooltipContent side="bottom" className="text-xs px-2 py-1">
          {tooltip}
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
}

function AppHeader() {
  const { resolvedTheme, setTheme } = useTheme();
  const isDark = resolvedTheme === "dark";

  return (
    <header className="flex h-12 shrink-0 items-center gap-2 border-b px-4">
      <SidebarTrigger className="-ml-1" />
      <div className="ml-auto flex items-center">
        {/* UI 控件区 */}
        <HeaderIconButton
          tooltip={isDark ? "切换浅色" : "切换深色"}
          onClick={() => setTheme(isDark ? "light" : "dark")}
        >
          <span className="relative flex items-center justify-center">
            <Sun
              className={`absolute transition-all duration-200 ${
                isDark ? "opacity-100 rotate-0 scale-100" : "opacity-0 rotate-90 scale-75"
              }`}
            />
            <Moon
              className={`transition-all duration-200 ${
                isDark ? "opacity-0 -rotate-90 scale-75" : "opacity-100 rotate-0 scale-100"
              }`}
            />
          </span>
        </HeaderIconButton>
        {/* 外链区 */}
        <div className="mx-2 h-4 w-px bg-border" />
        <TooltipProvider>
          <Tooltip>
            <TooltipTrigger asChild>
              <a
                href="https://github.com/openilink/openilink-hub"
                target="_blank"
                rel="noopener noreferrer"
                className="inline-flex items-center justify-center h-7 w-7 rounded-md text-foreground/80 transition-all duration-150 hover:text-foreground hover:bg-accent focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
              >
                <svg viewBox="0 0 24 24" className="size-[17px] fill-current">
                  <path d="M12 2C6.477 2 2 6.484 2 12.017c0 4.425 2.865 8.18 6.839 9.504.5.092.682-.217.682-.483 0-.237-.008-.868-.013-1.703-2.782.605-3.369-1.343-3.369-1.343-.454-1.158-1.11-1.466-1.11-1.466-.908-.62.069-.608.069-.608 1.003.07 1.531 1.032 1.531 1.032.892 1.53 2.341 1.088 2.91.832.092-.647.35-1.088.636-1.338-2.22-.253-4.555-1.113-4.555-4.951 0-1.093.39-1.988 1.029-2.688-.103-.253-.446-1.272.098-2.65 0 0 .84-.27 2.75 1.026A9.564 9.564 0 0112 6.844c.85.004 1.705.115 2.504.337 1.909-1.296 2.747-1.027 2.747-1.027.546 1.379.202 2.398.1 2.651.64.7 1.028 1.595 1.028 2.688 0 3.848-2.339 4.695-4.566 4.943.359.309.678.92.678 1.855 0 1.338-.012 2.419-.012 2.747 0 .268.18.58.688.482A10.019 10.019 0 0022 12.017C22 6.484 17.522 2 12 2z" />
                </svg>
              </a>
            </TooltipTrigger>
            <TooltipContent side="bottom" className="text-xs px-2 py-1">
              GitHub
            </TooltipContent>
          </Tooltip>
        </TooltipProvider>
      </div>
    </header>
  );
}

export function Layout() {
  const navigate = useNavigate();
  const location = useLocation();
  const [user, setUser] = useState<any>(null);

  useEffect(() => {
    api
      .me()
      .then(setUser)
      .catch(() => navigate("/login", { replace: true }));
  }, []);

  if (!user) return null;

  const isAdmin = user.role === "admin" || user.role === "superadmin";

  const roleLabel =
    user.role === "superadmin" ? "超级管理员" : user.role === "admin" ? "管理员" : "成员";

  async function handleLogout() {
    await api.logout();
    navigate("/login", { replace: true });
  }

  function isActive(path: string) {
    if (path === "/dashboard")
      return location.pathname === "/dashboard" || location.pathname.startsWith("/dashboard/bot/");
    if (path === "/dashboard/apps") return location.pathname.startsWith("/dashboard/apps");
    if (path === "/dashboard/admin/apps") return location.pathname === "/dashboard/admin/apps";
    return location.pathname === path;
  }

  return (
    <SidebarProvider>
      <Sidebar>
        {/* Logo */}
        <SidebarHeader>
          <SidebarMenu>
            <SidebarMenuItem>
              <SidebarMenuButton asChild tooltip="OpeniLink Hub">
                <Link to="/">
                  <LayoutDashboard />
                  <span className="font-semibold tracking-tight">OpeniLink Hub</span>
                </Link>
              </SidebarMenuButton>
            </SidebarMenuItem>
          </SidebarMenu>
        </SidebarHeader>

        <SidebarContent>
          {/* 主导航 */}
          <SidebarGroup>
            <SidebarGroupContent>
              <SidebarMenu>
                <SidebarMenuItem>
                  <SidebarMenuButton asChild isActive={isActive("/dashboard")} tooltip="Bot 管理">
                    <Link to="/dashboard">
                      <Bot />
                      <span>Bot 管理</span>
                    </Link>
                  </SidebarMenuButton>
                </SidebarMenuItem>
                <SidebarMenuItem>
                  <SidebarMenuButton
                    asChild
                    isActive={isActive("/dashboard/apps")}
                    tooltip="App 管理"
                  >
                    <Link to="/dashboard/apps">
                      <Blocks />
                      <span>App 管理</span>
                    </Link>
                  </SidebarMenuButton>
                </SidebarMenuItem>
              </SidebarMenu>
            </SidebarGroupContent>
          </SidebarGroup>

          {/* Webhook 插件 */}
          <SidebarGroup>
            <SidebarGroupLabel>Webhook 插件</SidebarGroupLabel>
            <SidebarGroupContent>
              <SidebarMenu>
                <SidebarMenuItem>
                  <SidebarMenuButton
                    asChild
                    isActive={isActive("/dashboard/webhook-plugins")}
                    tooltip="市场"
                  >
                    <Link to="/dashboard/webhook-plugins">
                      <Store />
                      <span>市场</span>
                    </Link>
                  </SidebarMenuButton>
                </SidebarMenuItem>
                <SidebarMenuItem>
                  <SidebarMenuButton
                    asChild
                    isActive={isActive("/dashboard/webhook-plugins/my")}
                    tooltip="我的插件"
                  >
                    <Link to="/dashboard/webhook-plugins/my">
                      <FolderOpen />
                      <span>我的插件</span>
                    </Link>
                  </SidebarMenuButton>
                </SidebarMenuItem>
                <SidebarMenuItem>
                  <SidebarMenuButton
                    asChild
                    isActive={isActive("/dashboard/webhook-plugins/debug")}
                    tooltip="调试器"
                  >
                    <Link to="/dashboard/webhook-plugins/debug">
                      <Bug />
                      <span>调试器</span>
                    </Link>
                  </SidebarMenuButton>
                </SidebarMenuItem>
                {isAdmin && (
                  <SidebarMenuItem>
                    <SidebarMenuButton
                      asChild
                      isActive={isActive("/dashboard/webhook-plugins/review")}
                      tooltip="审核"
                    >
                      <Link to="/dashboard/webhook-plugins/review">
                        <ShieldCheck />
                        <span>审核</span>
                      </Link>
                    </SidebarMenuButton>
                  </SidebarMenuItem>
                )}
              </SidebarMenu>
            </SidebarGroupContent>
          </SidebarGroup>

          {/* 系统管理（仅管理员） */}
          {isAdmin && (
            <SidebarGroup>
              <SidebarGroupLabel>系统管理</SidebarGroupLabel>
              <SidebarGroupContent>
                <SidebarMenu>
                  <SidebarMenuItem>
                    <SidebarMenuButton
                      asChild
                      isActive={isActive("/dashboard/admin")}
                      tooltip="概览"
                    >
                      <Link to="/dashboard/admin">
                        <BarChart3 />
                        <span>概览</span>
                      </Link>
                    </SidebarMenuButton>
                  </SidebarMenuItem>
                  <SidebarMenuItem>
                    <SidebarMenuButton
                      asChild
                      isActive={isActive("/dashboard/admin/users")}
                      tooltip="用户管理"
                    >
                      <Link to="/dashboard/admin/users">
                        <Users />
                        <span>用户管理</span>
                      </Link>
                    </SidebarMenuButton>
                  </SidebarMenuItem>
                  <SidebarMenuItem>
                    <SidebarMenuButton
                      asChild
                      isActive={isActive("/dashboard/admin/config")}
                      tooltip="系统配置"
                    >
                      <Link to="/dashboard/admin/config">
                        <Settings />
                        <span>系统配置</span>
                      </Link>
                    </SidebarMenuButton>
                  </SidebarMenuItem>
                  <SidebarMenuItem>
                    <SidebarMenuButton
                      asChild
                      isActive={isActive("/dashboard/admin/apps")}
                      tooltip="App 管理"
                    >
                      <Link to="/dashboard/admin/apps">
                        <Blocks />
                        <span>App 管理</span>
                      </Link>
                    </SidebarMenuButton>
                  </SidebarMenuItem>
                </SidebarMenu>
              </SidebarGroupContent>
            </SidebarGroup>
          )}
        </SidebarContent>

        {/* Footer：账号设置 + 用户菜单 */}
        <SidebarFooter>
          <SidebarMenu>
            <SidebarMenuItem>
              <SidebarMenuButton
                asChild
                isActive={isActive("/dashboard/settings")}
                tooltip="账号设置"
              >
                <Link to="/dashboard/settings">
                  <User />
                  <span>账号设置</span>
                </Link>
              </SidebarMenuButton>
            </SidebarMenuItem>

            <SidebarMenuItem>
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <SidebarMenuButton
                    size="lg"
                    tooltip={user.username}
                    className="data-[state=open]:bg-sidebar-accent data-[state=open]:text-sidebar-accent-foreground"
                  >
                    <Avatar className="size-8 rounded-lg shrink-0">
                      <AvatarFallback className="rounded-lg text-xs font-semibold bg-sidebar-primary text-sidebar-primary-foreground">
                        {user.username.charAt(0).toUpperCase()}
                      </AvatarFallback>
                    </Avatar>
                    <div className="flex flex-col gap-0.5 text-left leading-none min-w-0">
                      <span className="text-sm font-medium truncate">{user.username}</span>
                      <span className="text-xs text-muted-foreground truncate">{roleLabel}</span>
                    </div>
                    <ChevronsUpDown className="ml-auto shrink-0" />
                  </SidebarMenuButton>
                </DropdownMenuTrigger>
                <DropdownMenuContent side="top" align="end" className="w-56">
                  <DropdownMenuItem onClick={handleLogout}>
                    <LogOut />
                    退出登录
                  </DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
            </SidebarMenuItem>
          </SidebarMenu>
        </SidebarFooter>

        <SidebarRail />
      </Sidebar>

      <SidebarInset>
        <AppHeader />
        <main className="flex-1 overflow-auto">
          <div className="mx-auto max-w-6xl px-6 py-8 sm:px-8 sm:py-10 lg:px-10">
            <Outlet />
          </div>
        </main>
      </SidebarInset>
    </SidebarProvider>
  );
}
