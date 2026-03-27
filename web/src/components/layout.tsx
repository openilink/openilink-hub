import { Outlet, useNavigate, Link, useLocation } from "react-router-dom";
import { useEffect, useRef, useState } from "react";
import logoBlack from "@/assets/logo-black.svg";
import logoWhite from "@/assets/logo-white.svg";
import iconBlack from "@/assets/icon-black.svg";
import iconWhite from "@/assets/icon-white.svg";
import {
  LogOut,
  Github,
  Bot,
  ShieldCheck,
  Sun,
  Moon,
  ChevronsUpDown,
  Zap,
  Settings2,
  Search,
  MonitorDot,
  Puzzle,
  Circle,
  House,
} from "lucide-react";
import { api, botDisplayName } from "../lib/api";
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
  useSidebar,
} from "@/components/ui/sidebar";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
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
import { Input } from "@/components/ui/input";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip";
import * as React from "react";

function SidebarLogo() {
  const { open } = useSidebar();
  const { resolvedTheme } = useTheme();
  const isDark = resolvedTheme === "dark";
  return open ? (
    <img src={isDark ? logoWhite : logoBlack} alt="OpeniLink" className="h-7 w-auto" />
  ) : (
    <img src={isDark ? iconWhite : iconBlack} alt="OpeniLink" className="size-7" />
  );
}

const BREADCRUMB_LABELS: Record<string, string> = {
  accounts: "账号管理",
  apps: "应用",
  overview: "概览",
  settings: "设置",
  profile: "个人资料",
  security: "安全",
  admin: "系统管理",
  users: "用户管理",
  reviews: "审核中心",
  traces: "消息追踪",
};

const statusColors: Record<string, string> = {
  connected: "text-green-500 fill-green-500",
  disconnected: "text-muted-foreground fill-muted-foreground",
  error: "text-destructive fill-destructive",
  session_expired: "text-destructive fill-destructive",
};

function LayoutHeader() {
  const location = useLocation();
  const { resolvedTheme, setTheme } = useTheme();
  const searchRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === "k" && (e.metaKey || e.ctrlKey)) {
        e.preventDefault();
        searchRef.current?.focus();
      }
    };
    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, []);

  const pathSegments = location.pathname
    .split("/")
    .filter((s) => Boolean(s) && s !== "dashboard" && s !== "overview");
  const breadcrumbs = pathSegments.map((segment: string, index: number) => {
    const path = `/dashboard/${pathSegments.slice(0, index + 1).join("/")}`;
    let label = BREADCRUMB_LABELS[segment] || segment;
    if (segment.length > 20) label = "详情";
    return { label, path, isLast: index === pathSegments.length - 1 };
  });

  return (
    <header className="flex h-16 shrink-0 items-center justify-between gap-2 border-b bg-background/95 backdrop-blur px-6 sticky top-0 z-40">
      <div className="flex items-center gap-4">
        <SidebarTrigger className="-ml-2 h-9 w-9" />
        <Separator orientation="vertical" className="h-4 opacity-50" />
        <Breadcrumb>
          <BreadcrumbList>
            <BreadcrumbItem>
              <BreadcrumbLink asChild>
                <Link
                  to="/dashboard/overview"
                  className="flex items-center text-muted-foreground hover:text-primary transition-colors"
                >
                  <House className="h-3.5 w-3.5" />
                </Link>
              </BreadcrumbLink>
            </BreadcrumbItem>
            {breadcrumbs.length > 0 && <BreadcrumbSeparator className="opacity-30" />}
            {breadcrumbs.map((bc, i) => (
              <React.Fragment key={bc.path}>
                {i > 0 && <BreadcrumbSeparator className="hidden md:block opacity-30" />}
                <BreadcrumbItem>
                  {bc.isLast ? (
                    <BreadcrumbPage className="font-bold text-foreground">
                      {bc.label}
                    </BreadcrumbPage>
                  ) : (
                    <BreadcrumbLink asChild>
                      <Link
                        to={bc.path}
                        className="hover:text-primary transition-colors font-medium"
                      >
                        {bc.label}
                      </Link>
                    </BreadcrumbLink>
                  )}
                </BreadcrumbItem>
              </React.Fragment>
            ))}
          </BreadcrumbList>
        </Breadcrumb>
      </div>

      <TooltipProvider>
        <div className="flex items-center gap-3">
          <div className="hidden lg:flex relative items-center group">
            <Search className="absolute left-3 size-3.5 text-muted-foreground group-focus-within:text-primary transition-colors z-10" />
            <Input
              ref={searchRef}
              aria-label="搜索"
              placeholder="搜索..."
              className="h-9 w-56 pl-9 pr-14 focus:w-72 transition-all duration-200 bg-muted/40 border-border/50"
            />
            <kbd className="absolute right-2.5 pointer-events-none flex h-5 items-center gap-0.5 rounded border border-border/50 bg-muted px-1.5 text-[10px] font-medium text-muted-foreground group-focus-within:hidden">
              ⌘K
            </kbd>
          </div>
          <Tooltip>
            <TooltipTrigger asChild>
              <Button
                variant="ghost"
                size="icon"
                className="h-9 w-9"
                onClick={() => setTheme(resolvedTheme === "dark" ? "light" : "dark")}
              >
                {resolvedTheme === "dark" ? (
                  <Sun className="h-4 w-4" />
                ) : (
                  <Moon className="h-4 w-4" />
                )}
              </Button>
            </TooltipTrigger>
            <TooltipContent>切换外观主题</TooltipContent>
          </Tooltip>
          <Tooltip>
            <TooltipTrigger asChild>
              <Button variant="ghost" size="icon" className="h-9 w-9" asChild>
                <a
                  href="https://github.com/openilink/openilink-hub"
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  <Github className="h-4 w-4" />
                </a>
              </Button>
            </TooltipTrigger>
            <TooltipContent>GitHub 项目</TooltipContent>
          </Tooltip>
          <Tooltip>
            <TooltipTrigger asChild>
              <Button variant="ghost" size="icon" className="h-9 w-9 relative">
                <Zap className="h-4 w-4 text-yellow-500 fill-yellow-500/20" />
                <span className="absolute top-2 right-2 size-2 bg-primary rounded-full border-2 border-background animate-pulse" />
              </Button>
            </TooltipTrigger>
            <TooltipContent>活动</TooltipContent>
          </Tooltip>
        </div>
      </TooltipProvider>
    </header>
  );
}

export function Layout() {
  const navigate = useNavigate();
  const location = useLocation();
  const [user, setUser] = useState<any>(null);
  const [bots, setBots] = useState<any[]>([]);

  useEffect(() => {
    api
      .me()
      .then(setUser)
      .catch(() => navigate("/login", { replace: true }));
  }, []);

  useEffect(() => {
    if (user)
      api
        .listBots()
        .then((b) => setBots(b || []))
        .catch(() => {});
  }, [user]);

  // Refresh bot list when navigating back to accounts area
  useEffect(() => {
    if (user && location.pathname.startsWith("/dashboard/accounts")) {
      api
        .listBots()
        .then((b) => setBots(b || []))
        .catch(() => {});
    }
  }, [location.pathname]);

  if (!user) return null;

  const isAdmin = user.role === "admin" || user.role === "superadmin";

  // Logical matching for active states
  const isActive = (path: string) => location.pathname.startsWith(path);

  return (
    <SidebarProvider>
      <Sidebar variant="inset" collapsible="icon" className="border-r-0 shadow-none">
        <SidebarHeader className="h-16 justify-center">
          <SidebarMenu>
            <SidebarMenuItem>
              <SidebarMenuButton size="lg" asChild>
                <Link to="/dashboard/overview">
                  <SidebarLogo />
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
                  <SidebarMenuButton
                    asChild
                    isActive={location.pathname === "/dashboard/overview"}
                    tooltip="概览"
                  >
                    <Link to="/dashboard/overview">
                      <MonitorDot />
                      <span>概览</span>
                    </Link>
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
                      <SidebarMenuSubButton
                        asChild
                        size="sm"
                        isActive={location.pathname === "/dashboard/accounts"}
                      >
                        <Link to="/dashboard/accounts">全部账号</Link>
                      </SidebarMenuSubButton>
                    </SidebarMenuSubItem>
                    {bots.map((b) => (
                      <SidebarMenuSubItem key={b.id}>
                        <SidebarMenuSubButton
                          asChild
                          size="sm"
                          isActive={isActive(`/dashboard/accounts/${b.id}`)}
                        >
                          <Link to={`/dashboard/accounts/${b.id}`}>
                            <Circle
                              className={`size-2 ${statusColors[b.status] || "text-muted-foreground"}`}
                            />
                            <span className="truncate">{botDisplayName(b)}</span>
                          </Link>
                        </SidebarMenuSubButton>
                      </SidebarMenuSubItem>
                    ))}
                  </SidebarMenuSub>
                </SidebarMenuItem>
              </SidebarMenu>
            </SidebarGroupContent>
          </SidebarGroup>

          {/* 应用市场 */}
          <SidebarGroup>
            <SidebarGroupContent>
              <SidebarMenu>
                <SidebarMenuItem>
                  <SidebarMenuButton
                    asChild
                    isActive={isActive("/dashboard/apps")}
                    tooltip="应用市场"
                  >
                    <Link to="/dashboard/apps">
                      <Puzzle />
                      <span>应用市场</span>
                    </Link>
                  </SidebarMenuButton>
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
                        <SidebarMenuSubButton
                          asChild
                          size="sm"
                          isActive={location.pathname === "/dashboard/admin/overview"}
                        >
                          <Link to="/dashboard/admin/overview">系统概览</Link>
                        </SidebarMenuSubButton>
                      </SidebarMenuSubItem>
                      <SidebarMenuSubItem>
                        <SidebarMenuSubButton
                          asChild
                          size="sm"
                          isActive={isActive("/dashboard/admin/users")}
                        >
                          <Link to="/dashboard/admin/users">用户管理</Link>
                        </SidebarMenuSubButton>
                      </SidebarMenuSubItem>
                      <SidebarMenuSubItem>
                        <SidebarMenuSubButton
                          asChild
                          size="sm"
                          isActive={isActive("/dashboard/admin/reviews")}
                        >
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
              <SidebarMenuButton
                asChild
                isActive={isActive("/dashboard/settings")}
                tooltip="个人设置"
              >
                <Link to="/dashboard/settings/profile">
                  <Settings2 />
                  <span>偏好设置</span>
                </Link>
              </SidebarMenuButton>
            </SidebarMenuItem>
            <SidebarSeparator className="mx-0" />
            <SidebarMenuItem>
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <SidebarMenuButton
                    size="lg"
                    className="data-[state=open]:bg-sidebar-accent data-[state=open]:text-sidebar-accent-foreground"
                  >
                    <Avatar className="h-8 w-8 rounded-lg shadow-sm border border-border/50">
                      <AvatarFallback className="rounded-lg bg-primary/10 text-primary font-bold text-xs">
                        {user.username.charAt(0).toUpperCase()}
                      </AvatarFallback>
                    </Avatar>
                    <div className="grid flex-1 text-left text-sm leading-tight ml-1">
                      <span className="truncate font-semibold">{user.username}</span>
                      <span className="truncate text-[10px] text-muted-foreground font-medium uppercase">
                        {user.role}
                      </span>
                    </div>
                    <ChevronsUpDown className="ml-auto size-4 opacity-50" />
                  </SidebarMenuButton>
                </DropdownMenuTrigger>
                <DropdownMenuContent
                  className="w-(--radix-dropdown-menu-trigger-width) min-w-56 rounded-xl shadow-2xl"
                  side="top"
                  align="end"
                  sideOffset={8}
                >
                  <DropdownMenuItem
                    onClick={async () => {
                      await api.logout();
                      navigate("/login");
                    }}
                    className="cursor-pointer font-medium text-destructive focus:bg-destructive/10 focus:text-destructive"
                  >
                    <LogOut className="mr-2 h-4 w-4" />
                    退出登录
                  </DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
            </SidebarMenuItem>
          </SidebarMenu>
        </SidebarFooter>
        <SidebarRail />
      </Sidebar>

      <SidebarInset className="flex flex-col bg-background/50 rounded-tl-2xl overflow-hidden">
        <LayoutHeader />

        <main className="flex-1 overflow-y-auto overflow-x-hidden">
          <div className="h-full mx-auto w-full max-w-[1400px] p-6 lg:p-8 animate-in fade-in slide-in-from-bottom-2 duration-500">
            <Outlet />
          </div>
        </main>
      </SidebarInset>
    </SidebarProvider>
  );
}
