import { Outlet, useNavigate, Link, useLocation } from "react-router-dom";
import { useEffect, useState } from "react";
import {
  LogOut,
  Github,
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
  SidebarSeparator,
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

export function Layout() {
  const navigate = useNavigate();
  const location = useLocation();
  const [user, setUser] = useState<any>(null);
  const { resolvedTheme, setTheme } = useTheme();

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
      <Sidebar collapsible="icon">
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

          <SidebarSeparator />

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
            <>
              <SidebarSeparator />
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
            </>
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
            <SidebarSeparator className="mx-0" />
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
        <header className="flex h-12 shrink-0 items-center gap-2 border-b px-4">
          <SidebarTrigger className="-ml-1" />
          <div className="ml-auto flex items-center gap-1">
            <Button variant="ghost" size="icon-sm" asChild>
              <a
                href="https://github.com/openilink/openilink-hub"
                target="_blank"
                rel="noopener noreferrer"
              >
                <Github />
              </a>
            </Button>
            <Button
              variant="ghost"
              size="icon-sm"
              onClick={() => setTheme(resolvedTheme === "dark" ? "light" : "dark")}
            >
              {resolvedTheme === "dark" ? <Sun /> : <Moon />}
            </Button>
          </div>
        </header>
        <main className="flex-1 overflow-auto">
          <div className="mx-auto max-w-6xl px-6 py-8 sm:px-8 sm:py-10 lg:px-10">
            <Outlet />
          </div>
        </main>
      </SidebarInset>
    </SidebarProvider>
  );
}
