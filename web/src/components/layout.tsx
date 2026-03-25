import { Outlet, useNavigate, Link, useLocation } from "react-router-dom";
import { useEffect, useState } from "react";
import {
  LogOut,
  Github,
  Bot,
  ShieldCheck,
  Blocks,
  Sun,
  Moon,
  ChevronsUpDown,
  Cpu,
  Home,
  Zap,
  Settings2,
  Search,
  Layers,
  MonitorDot,
  BarChart3,
  Puzzle,
  Circle,
} from "lucide-react";
import { api } from "../lib/api";
import { useTheme } from "../lib/theme";
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarGroupContent,
  SidebarHeader,
  SidebarInset,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarMenuSub,
  SidebarMenuSubItem,
  SidebarMenuSubButton,
  SidebarProvider,
  SidebarRail,
  SidebarSeparator,
  SidebarTrigger,
} from "@/components/ui/sidebar";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from "@/components/ui/breadcrumb";
import { Separator } from "@/components/ui/separator";
import { Button } from "@/components/ui/button";
import * as React from "react";

const statusColors: Record<string, string> = {
  connected: "text-green-500 fill-green-500",
  disconnected: "text-muted-foreground fill-muted-foreground",
  error: "text-destructive fill-destructive",
  session_expired: "text-destructive fill-destructive",
};

export function Layout() {
  const navigate = useNavigate();
  const location = useLocation();
  const [user, setUser] = useState<any>(null);
  const [bots, setBots] = useState<any[]>([]);
  const { resolvedTheme, setTheme } = useTheme();

  useEffect(() => {
    api.me().then(setUser).catch(() => navigate("/login", { replace: true }));
  }, []);

  useEffect(() => {
    if (user) api.listBots().then(b => setBots(b || [])).catch(() => {});
  }, [user]);

  // Refresh bot list when navigating back to accounts area
  useEffect(() => {
    if (user && location.pathname.startsWith("/dashboard/accounts")) {
      api.listBots().then(b => setBots(b || [])).catch(() => {});
    }
  }, [location.pathname]);

  if (!user) return null;

  const isAdmin = user.role === "admin" || user.role === "superadmin";

  // Logical matching for active states
  const isActive = (path: string) => location.pathname.startsWith(path);

  // Business-driven Breadcrumbs mapping
  const pathSegments = location.pathname.split("/").filter(Boolean);
  const breadcrumbs = pathSegments.map((segment: string, index: number) => {
    const path = `/${pathSegments.slice(0, index + 1).join("/")}`;
    const labels: Record<string, string> = {
      dashboard: "控制台", accounts: "账号管理",
      apps: "应用", plugins: "插件", overview: "概览",
      marketplace: "市场", my: "我的",
      "webhook-plugins": "Webhook 插件", settings: "设置",
      profile: "个人资料", security: "安全",
      admin: "系统管理", users: "用户管理", reviews: "审核中心",
      channels: "转发规则", traces: "消息追踪",
    };
    let label = labels[segment] || segment;
    if (segment.length > 20) label = "详情"; // Handle IDs
    return { label, path, isLast: index === pathSegments.length - 1 };
  });

  return (
    <SidebarProvider>
      <Sidebar variant="inset" collapsible="icon" className="border-r-0 shadow-none">
        <SidebarHeader className="h-16 justify-center">
          <SidebarMenu>
            <SidebarMenuItem>
              <SidebarMenuButton size="lg" asChild>
                <Link to="/dashboard/overview">
                  <div className="flex aspect-square size-9 items-center justify-center rounded-xl bg-primary text-primary-foreground shadow-lg shadow-primary/20">
                    <Cpu className="size-5" />
                  </div>
                  <div className="grid flex-1 text-left text-sm leading-tight ml-1">
                    <span className="truncate font-bold text-base tracking-tight text-foreground">OpeniLink</span>
                  </div>
                </Link>
              </SidebarMenuButton>
            </SidebarMenuItem>
          </SidebarMenu>
        </SidebarHeader>

        <SidebarContent>
          {/* Overview */}
          <SidebarGroup>
            <SidebarGroupContent>
              <SidebarMenu>
                <SidebarMenuItem>
                  <SidebarMenuButton asChild isActive={location.pathname === "/dashboard/overview"} tooltip="概览">
                    <Link to="/dashboard/overview"><MonitorDot /><span>概览</span></Link>
                  </SidebarMenuButton>
                </SidebarMenuItem>
              </SidebarMenu>
            </SidebarGroupContent>
          </SidebarGroup>

          {/* 账号管理 */}
          <SidebarGroup>
            <SidebarGroupContent>
              <SidebarMenu>
                <SidebarMenuItem>
                  <SidebarMenuButton isActive={isActive("/dashboard/accounts")} tooltip="账号管理">
                    <Bot />
                    <span>账号管理</span>
                  </SidebarMenuButton>
                  <SidebarMenuSub>
                    <SidebarMenuSubItem>
                      <SidebarMenuSubButton asChild size="sm" isActive={location.pathname === "/dashboard/accounts"}>
                        <Link to="/dashboard/accounts">全部账号</Link>
                      </SidebarMenuSubButton>
                    </SidebarMenuSubItem>
                    {bots.map((b) => (
                      <SidebarMenuSubItem key={b.id}>
                        <SidebarMenuSubButton asChild size="sm" isActive={isActive(`/dashboard/accounts/${b.id}`)}>
                          <Link to={`/dashboard/accounts/${b.id}`}>
                            <Circle className={`size-2 ${statusColors[b.status] || "text-muted-foreground"}`} />
                            <span className="truncate">{b.name}</span>
                          </Link>
                        </SidebarMenuSubButton>
                      </SidebarMenuSubItem>
                    ))}
                  </SidebarMenuSub>
                </SidebarMenuItem>
              </SidebarMenu>
            </SidebarGroupContent>
          </SidebarGroup>

          {/* 扩展 */}
          <SidebarGroup>
            <SidebarGroupContent>
              <SidebarMenu>
                <SidebarMenuItem>
                  <SidebarMenuButton isActive={isActive("/dashboard/apps") || isActive("/dashboard/plugins")} tooltip="扩展">
                    <Puzzle />
                    <span>扩展</span>
                  </SidebarMenuButton>
                  <SidebarMenuSub>
                    <SidebarMenuSubItem>
                      <SidebarMenuSubButton asChild size="sm" isActive={isActive("/dashboard/apps/marketplace")}>
                        <Link to="/dashboard/apps/marketplace">应用市场</Link>
                      </SidebarMenuSubButton>
                    </SidebarMenuSubItem>
                    <SidebarMenuSubItem>
                      <SidebarMenuSubButton asChild size="sm" isActive={location.pathname === "/dashboard/apps/my" || location.pathname === "/dashboard/apps"}>
                        <Link to="/dashboard/apps/my">我的应用</Link>
                      </SidebarMenuSubButton>
                    </SidebarMenuSubItem>
                    <SidebarMenuSubItem>
                      <SidebarMenuSubButton asChild size="sm" isActive={isActive("/dashboard/plugins/marketplace") || location.pathname === "/dashboard/plugins"}>
                        <Link to="/dashboard/plugins/marketplace">插件市场</Link>
                      </SidebarMenuSubButton>
                    </SidebarMenuSubItem>
                    <SidebarMenuSubItem>
                      <SidebarMenuSubButton asChild size="sm" isActive={isActive("/dashboard/plugins/my")}>
                        <Link to="/dashboard/plugins/my">我的插件</Link>
                      </SidebarMenuSubButton>
                    </SidebarMenuSubItem>
                  </SidebarMenuSub>
                </SidebarMenuItem>
              </SidebarMenu>
            </SidebarGroupContent>
          </SidebarGroup>

          {/* 管理 — admin only */}
          {isAdmin && (
            <SidebarGroup>
              <SidebarGroupContent>
                <SidebarMenu>
                  <SidebarMenuItem>
                    <SidebarMenuButton isActive={isActive("/dashboard/admin")} tooltip="管理">
                      <ShieldCheck />
                      <span>管理</span>
                    </SidebarMenuButton>
                    <SidebarMenuSub>
                      <SidebarMenuSubItem>
                        <SidebarMenuSubButton asChild size="sm" isActive={location.pathname === "/dashboard/admin/overview"}>
                          <Link to="/dashboard/admin/overview">系统概览</Link>
                        </SidebarMenuSubButton>
                      </SidebarMenuSubItem>
                      <SidebarMenuSubItem>
                        <SidebarMenuSubButton asChild size="sm" isActive={isActive("/dashboard/admin/users")}>
                          <Link to="/dashboard/admin/users">用户管理</Link>
                        </SidebarMenuSubButton>
                      </SidebarMenuSubItem>
                      <SidebarMenuSubItem>
                        <SidebarMenuSubButton asChild size="sm" isActive={isActive("/dashboard/admin/reviews")}>
                          <Link to="/dashboard/admin/reviews">审核中心</Link>
                        </SidebarMenuSubButton>
                      </SidebarMenuSubItem>
                    </SidebarMenuSub>
                  </SidebarMenuItem>
                </SidebarMenu>
              </SidebarGroupContent>
            </SidebarGroup>
          )}
        </SidebarContent>

        <SidebarFooter>
          <SidebarMenu>
            <SidebarMenuItem>
              <SidebarMenuButton asChild isActive={isActive("/dashboard/settings")} tooltip="个人设置">
                <Link to="/dashboard/settings/profile"><Settings2 /><span>偏好设置</span></Link>
              </SidebarMenuButton>
            </SidebarMenuItem>
            <SidebarSeparator className="mx-0" />
            <SidebarMenuItem>
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <SidebarMenuButton size="lg" className="data-[state=open]:bg-sidebar-accent data-[state=open]:text-sidebar-accent-foreground">
                    <Avatar className="h-8 w-8 rounded-lg shadow-sm border border-border/50">
                      <AvatarFallback className="rounded-lg bg-primary/10 text-primary font-bold text-xs">
                        {user.username.charAt(0).toUpperCase()}
                      </AvatarFallback>
                    </Avatar>
                    <div className="grid flex-1 text-left text-sm leading-tight ml-1">
                      <span className="truncate font-semibold">{user.username}</span>
                      <span className="truncate text-[10px] text-muted-foreground font-medium uppercase">{user.role}</span>
                    </div>
                    <ChevronsUpDown className="ml-auto size-4 opacity-50" />
                  </SidebarMenuButton>
                </DropdownMenuTrigger>
                <DropdownMenuContent className="w-(--radix-dropdown-menu-trigger-width) min-w-56 rounded-xl shadow-2xl" side="top" align="end" sideOffset={8}>
                  <DropdownMenuItem asChild><a href="https://github.com/openilink/openilink-hub" target="_blank" className="cursor-pointer font-medium"><Github className="mr-2 h-4 w-4" />GitHub 项目</a></DropdownMenuItem>
                  <DropdownMenuItem onClick={() => setTheme(resolvedTheme === "dark" ? "light" : "dark")} className="cursor-pointer font-medium">{resolvedTheme === "dark" ? <Sun className="mr-2 h-4 w-4" /> : <Moon className="mr-2 h-4 w-4" />}切换外观主题</DropdownMenuItem>
                  <DropdownMenuSeparator />
                  <DropdownMenuItem onClick={async () => { await api.logout(); navigate("/login"); }} className="cursor-pointer font-medium text-destructive focus:bg-destructive/10 focus:text-destructive"><LogOut className="mr-2 h-4 w-4" />退出登录</DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
            </SidebarMenuItem>
          </SidebarMenu>
        </SidebarFooter>
        <SidebarRail />
      </Sidebar>

      <SidebarInset className="flex flex-col bg-background/50">
        <header className="flex h-16 shrink-0 items-center justify-between gap-2 border-b bg-background/95 backdrop-blur px-6 sticky top-0 z-40">
          <div className="flex items-center gap-4">
            <SidebarTrigger className="-ml-2 h-9 w-9" />
            <Separator orientation="vertical" className="h-4 opacity-50" />
            <Breadcrumb>
              <BreadcrumbList>
                <BreadcrumbItem className="hidden md:block">
                  <BreadcrumbLink asChild>
                    <Link to="/dashboard/overview" className="hover:text-primary transition-colors">
                      <Home className="h-4 w-4" />
                    </Link>
                  </BreadcrumbLink>
                </BreadcrumbItem>
                {breadcrumbs.map((bc, i) => (
                  <React.Fragment key={bc.path}>
                    <BreadcrumbSeparator className="hidden md:block opacity-30" />
                    <BreadcrumbItem>
                      {bc.isLast ? (
                        <BreadcrumbPage className="font-bold text-foreground">{bc.label}</BreadcrumbPage>
                      ) : (
                        <BreadcrumbLink asChild>
                          <Link to={bc.path} className="hover:text-primary transition-colors font-medium">{bc.label}</Link>
                        </BreadcrumbLink>
                      )}
                    </BreadcrumbItem>
                  </React.Fragment>
                ))}
              </BreadcrumbList>
            </Breadcrumb>
          </div>

          <div className="flex items-center gap-3">
             <div className="hidden lg:flex relative items-center group">
                <Search className="absolute left-3 size-3.5 text-muted-foreground group-focus-within:text-primary transition-colors" />
                <input placeholder="搜索..." className="h-9 w-64 rounded-full bg-muted/50 border-transparent pl-9 pr-4 text-xs font-medium focus:bg-background focus:border-border transition-all outline-none" />
             </div>
             <Button variant="ghost" size="icon" className="h-9 w-9 rounded-full relative">
                <Zap className="h-4 w-4 text-yellow-500 fill-yellow-500/20" />
                <span className="absolute top-2 right-2 size-2 bg-primary rounded-full border-2 border-background animate-pulse" />
             </Button>
          </div>
        </header>

        <main className="flex-1 overflow-y-auto overflow-x-hidden">
          <div className="mx-auto w-full max-w-[1400px] p-6 lg:p-8 animate-in fade-in slide-in-from-bottom-2 duration-500">
            <Outlet />
          </div>
        </main>
      </SidebarInset>
    </SidebarProvider>
  );
}
