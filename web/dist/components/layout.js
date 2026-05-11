import { jsx as _jsx, jsxs as _jsxs, Fragment as _Fragment } from "react/jsx-runtime";
import { Outlet, useNavigate, Link, useLocation } from "react-router-dom";
import { useEffect, useRef, useState } from "react";
import { useUser } from "@/hooks/use-auth";
import { useBots } from "@/hooks/use-bots";
import { useAuthStore } from "@/stores/auth-store";
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
  Code2,
  ShieldAlert,
  X,
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
  DropdownMenuLabel,
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
import { Input } from "@/components/ui/input";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip";
import * as React from "react";
function SidebarLogo() {
  const { open } = useSidebar();
  const { resolvedTheme } = useTheme();
  const isDark = resolvedTheme === "dark";
  return open
    ? _jsx("img", {
        src: isDark ? logoWhite : logoBlack,
        alt: "OpeniLink",
        className: "h-7 w-auto",
      })
    : _jsx("img", { src: isDark ? iconWhite : iconBlack, alt: "OpeniLink", className: "size-7" });
}
const BREADCRUMB_LABELS = {
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
  developer: "开发者",
  install: "安装应用",
  console: "控制台",
  onboarding: "引导",
};
// Intermediate-only segments that are NOT standalone routes.
// These are skipped in breadcrumbs when followed by a dynamic ID.
const BREADCRUMB_SKIP = new Set(["apps", "install"]);
const statusColors = {
  connected: "text-green-500 fill-green-500",
  disconnected: "text-muted-foreground fill-muted-foreground",
  error: "text-destructive fill-destructive",
  session_expired: "text-destructive fill-destructive",
};
function LayoutHeader() {
  const location = useLocation();
  const { resolvedTheme, setTheme } = useTheme();
  const searchRef = useRef(null);
  useEffect(() => {
    const handleKeyDown = (e) => {
      if (e.key === "k" && (e.metaKey || e.ctrlKey)) {
        e.preventDefault();
        searchRef.current?.focus();
      }
    };
    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, []);
  // Build breadcrumbs from path, skipping segments that are intermediate
  // parts of a compound route (e.g. "apps" in /accounts/:id/apps/:iid).
  const rawSegments = location.pathname
    .split("/")
    .filter((s) => Boolean(s) && s !== "dashboard" && s !== "overview");
  const breadcrumbs = [];
  for (let i = 0; i < rawSegments.length; i++) {
    const segment = rawSegments[i];
    const path = `/dashboard/${rawSegments.slice(0, i + 1).join("/")}`;
    // Skip intermediate-only segments (e.g. "apps" in /accounts/:id/apps/:iid)
    if (BREADCRUMB_SKIP.has(segment) && i > 0 && i < rawSegments.length - 1) {
      continue;
    }
    let label = BREADCRUMB_LABELS[segment] || segment;
    if (segment.length > 20) label = "详情";
    breadcrumbs.push({ label, path, isLast: i === rawSegments.length - 1 });
  }
  return _jsxs("header", {
    className:
      "flex h-16 shrink-0 items-center justify-between gap-2 border-b bg-background/95 backdrop-blur px-6 sticky top-0 z-40",
    children: [
      _jsxs("div", {
        className: "flex items-center gap-4",
        children: [
          _jsx(SidebarTrigger, { className: "-ml-2 h-9 w-9" }),
          _jsx(Separator, { orientation: "vertical", className: "h-4 opacity-50" }),
          _jsx(Breadcrumb, {
            children: _jsxs(BreadcrumbList, {
              children: [
                _jsx(BreadcrumbItem, {
                  children: _jsx(BreadcrumbLink, {
                    asChild: true,
                    children: _jsx(Link, {
                      to: "/dashboard/overview",
                      className:
                        "flex items-center text-muted-foreground hover:text-primary transition-colors",
                      children: _jsx(House, { className: "h-3.5 w-3.5" }),
                    }),
                  }),
                }),
                breadcrumbs.length > 0 && _jsx(BreadcrumbSeparator, { className: "opacity-30" }),
                breadcrumbs.map((bc, i) =>
                  _jsxs(
                    React.Fragment,
                    {
                      children: [
                        i > 0 && _jsx(BreadcrumbSeparator, { className: "opacity-30" }),
                        _jsx(BreadcrumbItem, {
                          children: bc.isLast
                            ? _jsx(BreadcrumbPage, {
                                className: "font-bold text-foreground",
                                children: bc.label,
                              })
                            : _jsx(BreadcrumbLink, {
                                asChild: true,
                                children: _jsx(Link, {
                                  to: bc.path,
                                  className: "hover:text-primary transition-colors font-medium",
                                  children: bc.label,
                                }),
                              }),
                        }),
                      ],
                    },
                    bc.path,
                  ),
                ),
              ],
            }),
          }),
        ],
      }),
      _jsx(TooltipProvider, {
        children: _jsxs("div", {
          className: "flex items-center gap-3",
          children: [
            _jsxs("div", {
              className: "hidden lg:flex relative items-center group",
              children: [
                _jsx(Search, {
                  className:
                    "absolute left-3 size-3.5 text-muted-foreground group-focus-within:text-primary transition-colors z-10",
                }),
                _jsx(Input, {
                  ref: searchRef,
                  "aria-label": "\u641C\u7D22",
                  placeholder: "\u641C\u7D22...",
                  className:
                    "h-9 w-56 pl-9 pr-14 focus:w-72 transition-all duration-200 bg-muted/40 border-border/50",
                }),
                _jsx("kbd", {
                  className:
                    "absolute right-2.5 pointer-events-none flex h-5 items-center gap-0.5 rounded border border-border/50 bg-muted px-1.5 text-[10px] font-medium text-muted-foreground group-focus-within:hidden",
                  children: "\u2318K",
                }),
              ],
            }),
            _jsxs(Tooltip, {
              children: [
                _jsx(TooltipTrigger, {
                  asChild: true,
                  children: _jsx(Button, {
                    variant: "ghost",
                    size: "icon",
                    className: "h-9 w-9",
                    onClick: () => setTheme(resolvedTheme === "dark" ? "light" : "dark"),
                    children:
                      resolvedTheme === "dark"
                        ? _jsx(Sun, { className: "h-4 w-4" })
                        : _jsx(Moon, { className: "h-4 w-4" }),
                  }),
                }),
                _jsx(TooltipContent, { children: "\u5207\u6362\u5916\u89C2\u4E3B\u9898" }),
              ],
            }),
            _jsxs(Tooltip, {
              children: [
                _jsx(TooltipTrigger, {
                  asChild: true,
                  children: _jsx(Button, {
                    variant: "ghost",
                    size: "icon",
                    className: "h-9 w-9",
                    asChild: true,
                    children: _jsx("a", {
                      href: "https://github.com/openilink/openilink-hub",
                      target: "_blank",
                      rel: "noopener noreferrer",
                      children: _jsx(Github, { className: "h-4 w-4" }),
                    }),
                  }),
                }),
                _jsx(TooltipContent, { children: "GitHub \u9879\u76EE" }),
              ],
            }),
            _jsxs(Tooltip, {
              children: [
                _jsx(TooltipTrigger, {
                  asChild: true,
                  children: _jsxs(Button, {
                    variant: "ghost",
                    size: "icon",
                    className: "h-9 w-9 relative",
                    children: [
                      _jsx(Zap, { className: "h-4 w-4 text-yellow-500 fill-yellow-500/20" }),
                      _jsx("span", {
                        className:
                          "absolute top-2 right-2 size-2 bg-primary rounded-full border-2 border-background animate-pulse",
                      }),
                    ],
                  }),
                }),
                _jsx(TooltipContent, { children: "\u6D3B\u52A8" }),
              ],
            }),
          ],
        }),
      }),
    ],
  });
}
function SecurityBanner() {
  const [dismissed, setDismissed] = useState(false);
  if (dismissed) return null;
  return _jsxs("div", {
    className:
      "flex items-center gap-3 px-6 py-2.5 bg-amber-500/10 border-b border-amber-500/20 text-sm",
    children: [
      _jsx(ShieldAlert, { className: "h-4 w-4 text-amber-500 shrink-0" }),
      _jsxs("span", {
        className: "text-amber-700 dark:text-amber-400",
        children: [
          "\u60A8\u7684\u8D26\u53F7\u5C1A\u672A\u8BBE\u7F6E\u5BC6\u7801\u6216\u7ED1\u5B9A\u901A\u884C\u5BC6\u94A5\uFF0C\u767B\u51FA\u540E\u53EF\u80FD\u65E0\u6CD5\u518D\u6B21\u767B\u5F55\u3002",
          _jsx(Link, {
            to: "/dashboard/settings/security",
            className: "underline font-medium ml-1 hover:no-underline",
            children: "\u524D\u5F80\u8BBE\u7F6E",
          }),
        ],
      }),
      _jsx("button", {
        onClick: () => setDismissed(true),
        className: "ml-auto text-amber-500/60 hover:text-amber-500",
        children: _jsx(X, { className: "h-3.5 w-3.5" }),
      }),
    ],
  });
}
export function Layout() {
  const navigate = useNavigate();
  const location = useLocation();
  const { data: user, isError } = useUser();
  const { data: bots = [] } = useBots();
  const [version, setVersion] = useState("");
  useEffect(() => {
    if (isError) navigate("/login", { replace: true });
  }, [isError, navigate]);
  useEffect(() => {
    api
      .info()
      .then((data) => setVersion(data.version || ""))
      .catch(() => {});
  }, []);
  if (!user) return null;
  const isAdmin = user.role === "admin" || user.role === "superadmin";
  // Logical matching for active states
  const isActive = (path) => location.pathname.startsWith(path);
  return _jsxs(SidebarProvider, {
    children: [
      _jsxs(Sidebar, {
        variant: "inset",
        collapsible: "icon",
        className: "border-r-0 shadow-none",
        children: [
          _jsx(SidebarHeader, {
            className: "h-16 justify-center",
            children: _jsx(SidebarMenu, {
              children: _jsx(SidebarMenuItem, {
                children: _jsx(SidebarMenuButton, {
                  size: "lg",
                  asChild: true,
                  children: _jsx(Link, {
                    to: "/dashboard/overview",
                    children: _jsx(SidebarLogo, {}),
                  }),
                }),
              }),
            }),
          }),
          _jsxs(SidebarContent, {
            children: [
              _jsx(SidebarGroup, {
                children: _jsx(SidebarGroupContent, {
                  children: _jsx(SidebarMenu, {
                    children: _jsx(SidebarMenuItem, {
                      children: _jsx(SidebarMenuButton, {
                        asChild: true,
                        isActive: location.pathname === "/dashboard/overview",
                        tooltip: "\u6982\u89C8",
                        children: _jsxs(Link, {
                          to: "/dashboard/overview",
                          children: [
                            _jsx(MonitorDot, {}),
                            _jsx("span", { children: "\u6982\u89C8" }),
                          ],
                        }),
                      }),
                    }),
                  }),
                }),
              }),
              _jsx(SidebarGroup, {
                children: _jsx(SidebarGroupContent, {
                  children: _jsx(SidebarMenu, {
                    children: _jsxs(SidebarMenuItem, {
                      children: [
                        _jsxs(SidebarMenuButton, {
                          isActive: isActive("/dashboard/accounts"),
                          tooltip: "\u8D26\u53F7\u7BA1\u7406",
                          children: [
                            _jsx(Bot, {}),
                            _jsx("span", { children: "\u8D26\u53F7\u7BA1\u7406" }),
                          ],
                        }),
                        _jsxs(SidebarMenuSub, {
                          children: [
                            _jsx(SidebarMenuSubItem, {
                              children: _jsx(SidebarMenuSubButton, {
                                asChild: true,
                                size: "sm",
                                isActive: location.pathname === "/dashboard/accounts",
                                children: _jsx(Link, {
                                  to: "/dashboard/accounts",
                                  children: "\u5168\u90E8\u8D26\u53F7",
                                }),
                              }),
                            }),
                            bots.map((b) =>
                              _jsx(
                                SidebarMenuSubItem,
                                {
                                  children: _jsx(SidebarMenuSubButton, {
                                    asChild: true,
                                    size: "sm",
                                    isActive: isActive(`/dashboard/accounts/${b.id}`),
                                    children: _jsxs(Link, {
                                      to: `/dashboard/accounts/${b.id}`,
                                      children: [
                                        _jsx(Circle, {
                                          className: `size-2 ${statusColors[b.status] || "text-muted-foreground"}`,
                                        }),
                                        _jsx("span", {
                                          className: "truncate",
                                          children: botDisplayName(b),
                                        }),
                                      ],
                                    }),
                                  }),
                                },
                                b.id,
                              ),
                            ),
                          ],
                        }),
                      ],
                    }),
                  }),
                }),
              }),
              _jsx(SidebarGroup, {
                children: _jsx(SidebarGroupContent, {
                  children: _jsx(SidebarMenu, {
                    children: _jsx(SidebarMenuItem, {
                      children: _jsx(SidebarMenuButton, {
                        asChild: true,
                        isActive: isActive("/dashboard/apps"),
                        tooltip: "\u5E94\u7528\u5E02\u573A",
                        children: _jsxs(Link, {
                          to: "/dashboard/apps",
                          children: [
                            _jsx(Puzzle, {}),
                            _jsx("span", { children: "\u5E94\u7528\u5E02\u573A" }),
                          ],
                        }),
                      }),
                    }),
                  }),
                }),
              }),
              _jsx(SidebarGroup, {
                children: _jsx(SidebarGroupContent, {
                  children: _jsx(SidebarMenu, {
                    children: _jsx(SidebarMenuItem, {
                      children: _jsx(SidebarMenuButton, {
                        asChild: true,
                        isActive: isActive("/dashboard/developer"),
                        tooltip: "\u5F00\u53D1\u8005",
                        children: _jsxs(Link, {
                          to: "/dashboard/developer/apps",
                          children: [
                            _jsx(Code2, {}),
                            _jsx("span", { children: "\u5F00\u53D1\u8005" }),
                          ],
                        }),
                      }),
                    }),
                  }),
                }),
              }),
              isAdmin &&
                _jsx(SidebarGroup, {
                  children: _jsx(SidebarGroupContent, {
                    children: _jsx(SidebarMenu, {
                      children: _jsxs(SidebarMenuItem, {
                        children: [
                          _jsxs(SidebarMenuButton, {
                            isActive: isActive("/dashboard/admin"),
                            tooltip: "\u7BA1\u7406",
                            children: [
                              _jsx(ShieldCheck, {}),
                              _jsx("span", { children: "\u7BA1\u7406" }),
                            ],
                          }),
                          _jsxs(SidebarMenuSub, {
                            children: [
                              _jsx(SidebarMenuSubItem, {
                                children: _jsx(SidebarMenuSubButton, {
                                  asChild: true,
                                  size: "sm",
                                  isActive: location.pathname === "/dashboard/admin/overview",
                                  children: _jsx(Link, {
                                    to: "/dashboard/admin/overview",
                                    children: "\u7CFB\u7EDF\u6982\u89C8",
                                  }),
                                }),
                              }),
                              _jsx(SidebarMenuSubItem, {
                                children: _jsx(SidebarMenuSubButton, {
                                  asChild: true,
                                  size: "sm",
                                  isActive: isActive("/dashboard/admin/users"),
                                  children: _jsx(Link, {
                                    to: "/dashboard/admin/users",
                                    children: "\u7528\u6237\u7BA1\u7406",
                                  }),
                                }),
                              }),
                              _jsx(SidebarMenuSubItem, {
                                children: _jsx(SidebarMenuSubButton, {
                                  asChild: true,
                                  size: "sm",
                                  isActive: isActive("/dashboard/admin/reviews"),
                                  children: _jsx(Link, {
                                    to: "/dashboard/admin/reviews",
                                    children: "\u5BA1\u6838\u4E2D\u5FC3",
                                  }),
                                }),
                              }),
                            ],
                          }),
                        ],
                      }),
                    }),
                  }),
                }),
            ],
          }),
          _jsx(SidebarFooter, {
            children: _jsxs(SidebarMenu, {
              children: [
                _jsx(SidebarMenuItem, {
                  children: _jsx(SidebarMenuButton, {
                    asChild: true,
                    isActive: isActive("/dashboard/settings"),
                    tooltip: "\u4E2A\u4EBA\u8BBE\u7F6E",
                    children: _jsxs(Link, {
                      to: "/dashboard/settings/profile",
                      children: [
                        _jsx(Settings2, {}),
                        _jsx("span", { children: "\u504F\u597D\u8BBE\u7F6E" }),
                      ],
                    }),
                  }),
                }),
                _jsx(SidebarSeparator, { className: "mx-0" }),
                _jsx(SidebarMenuItem, {
                  children: _jsxs(DropdownMenu, {
                    children: [
                      _jsx(DropdownMenuTrigger, {
                        asChild: true,
                        children: _jsxs(SidebarMenuButton, {
                          size: "lg",
                          className:
                            "data-[state=open]:bg-sidebar-accent data-[state=open]:text-sidebar-accent-foreground",
                          children: [
                            _jsx(Avatar, {
                              className: "h-8 w-8 rounded-lg shadow-sm border border-border/50",
                              children: _jsx(AvatarFallback, {
                                className:
                                  "rounded-lg bg-primary/10 text-primary font-bold text-xs",
                                children: user.username.charAt(0).toUpperCase(),
                              }),
                            }),
                            _jsxs("div", {
                              className: "grid flex-1 text-left text-sm leading-tight ml-1",
                              children: [
                                _jsx("span", {
                                  className: "truncate font-semibold",
                                  children: user.username,
                                }),
                                _jsx("span", {
                                  className:
                                    "truncate text-[10px] text-muted-foreground font-medium uppercase",
                                  children: user.role,
                                }),
                              ],
                            }),
                            _jsx(ChevronsUpDown, { className: "ml-auto size-4 opacity-50" }),
                          ],
                        }),
                      }),
                      _jsxs(DropdownMenuContent, {
                        className:
                          "w-(--radix-dropdown-menu-trigger-width) min-w-56 rounded-xl shadow-2xl",
                        side: "top",
                        align: "end",
                        sideOffset: 8,
                        children: [
                          _jsxs(DropdownMenuItem, {
                            onClick: async () => {
                              await useAuthStore.getState().logout();
                              navigate("/login");
                            },
                            className:
                              "cursor-pointer font-medium text-destructive focus:bg-destructive/10 focus:text-destructive",
                            children: [
                              _jsx(LogOut, { className: "mr-2 h-4 w-4" }),
                              "\u9000\u51FA\u767B\u5F55",
                            ],
                          }),
                          version &&
                            _jsxs(_Fragment, {
                              children: [
                                _jsx(DropdownMenuSeparator, {}),
                                _jsx(DropdownMenuLabel, {
                                  className:
                                    "text-[10px] text-muted-foreground text-center font-normal",
                                  children: /^\d/.test(version) ? `v${version}` : version,
                                }),
                              ],
                            }),
                        ],
                      }),
                    ],
                  }),
                }),
              ],
            }),
          }),
          _jsx(SidebarRail, {}),
        ],
      }),
      _jsxs(SidebarInset, {
        className: "flex flex-col bg-background/50 rounded-tl-2xl overflow-hidden",
        children: [
          _jsx(LayoutHeader, {}),
          user &&
            !user.has_password &&
            !user.has_passkey &&
            !user.has_oauth &&
            _jsx(SecurityBanner, {}),
          _jsx("main", {
            className:
              "flex-1 overflow-y-auto overflow-x-hidden [&:has([data-full-page])]:overflow-hidden",
            children: _jsx("div", {
              className:
                "h-full mx-auto w-full max-w-[1400px] p-6 lg:p-8 animate-in fade-in slide-in-from-bottom-2 duration-500 [&:has([data-full-page])]:p-0 [&:has([data-full-page])]:max-w-none",
              children: _jsx(Outlet, {}),
            }),
          }),
        ],
      }),
    ],
  });
}
