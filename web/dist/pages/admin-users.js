import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { MoreVertical, Check, X, Trash2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { useToast } from "@/hooks/use-toast";
import { useConfirm } from "@/components/ui/confirm-dialog";
import { useAdminUsers, useUpdateUserStatus, useDeleteUser } from "@/hooks/use-admin";
export function AdminUsersPage() {
  const { data: users = [], isLoading: loading } = useAdminUsers();
  const updateStatusMutation = useUpdateUserStatus();
  const deleteUserMutation = useDeleteUser();
  const { toast } = useToast();
  const { confirm, ConfirmDialog } = useConfirm();
  async function handleUpdateStatus(id, status) {
    try {
      await updateStatusMutation.mutateAsync({ id, status });
      toast({ title: "状态已更新" });
    } catch (e) {
      toast({ variant: "destructive", title: "操作失败", description: e.message });
    }
  }
  return _jsxs("div", {
    className: "space-y-6",
    children: [
      ConfirmDialog,
      _jsx("div", {
        className: "flex items-start justify-between gap-4",
        children: _jsxs("div", {
          children: [
            _jsx("h1", {
              className: "text-2xl font-bold tracking-tight",
              children: "\u7528\u6237\u7BA1\u7406",
            }),
            _jsx("p", {
              className: "text-sm text-muted-foreground mt-0.5",
              children: "\u7BA1\u7406\u5E73\u53F0\u7528\u6237\u8D26\u53F7\u3002",
            }),
          ],
        }),
      }),
      _jsx("div", {
        className: "rounded-xl border border-border/50 overflow-hidden",
        children: _jsxs(Table, {
          children: [
            _jsx(TableHeader, {
              className: "bg-muted/30",
              children: _jsxs(TableRow, {
                children: [
                  _jsx(TableHead, { children: "\u7528\u6237\u540D" }),
                  _jsx(TableHead, { children: "\u89D2\u8272" }),
                  _jsx(TableHead, { children: "\u72B6\u6001" }),
                  _jsx(TableHead, { children: "\u6CE8\u518C\u65F6\u95F4" }),
                  _jsx(TableHead, { className: "text-right", children: "\u64CD\u4F5C" }),
                ],
              }),
            }),
            _jsx(TableBody, {
              children: loading
                ? [1, 2, 3].map((i) =>
                    _jsx(
                      TableRow,
                      {
                        children: _jsx(TableCell, {
                          colSpan: 5,
                          children: _jsx("div", {
                            className: "h-4 w-full bg-muted animate-pulse rounded",
                          }),
                        }),
                      },
                      i,
                    ),
                  )
                : users.map((u) =>
                    _jsxs(
                      TableRow,
                      {
                        className: "group",
                        children: [
                          _jsx(TableCell, { className: "font-bold", children: u.username }),
                          _jsx(TableCell, {
                            children: _jsx(Badge, {
                              variant: "secondary",
                              className: "uppercase text-[9px] font-black",
                              children: u.role,
                            }),
                          }),
                          _jsx(TableCell, {
                            children: _jsx(Badge, {
                              variant: u.status === "active" ? "default" : "outline",
                              className: "h-5",
                              children: u.status,
                            }),
                          }),
                          _jsx(TableCell, {
                            className: "text-xs text-muted-foreground",
                            children: new Date(u.created_at * 1000).toLocaleDateString(),
                          }),
                          _jsx(TableCell, {
                            className: "text-right",
                            children: _jsxs(DropdownMenu, {
                              children: [
                                _jsx(DropdownMenuTrigger, {
                                  asChild: true,
                                  children: _jsx(Button, {
                                    variant: "ghost",
                                    size: "icon",
                                    className: "h-8 w-8 rounded-full",
                                    children: _jsx(MoreVertical, { className: "h-4 w-4" }),
                                  }),
                                }),
                                _jsxs(DropdownMenuContent, {
                                  align: "end",
                                  className: "rounded-xl",
                                  children: [
                                    _jsxs(DropdownMenuItem, {
                                      onClick: () =>
                                        handleUpdateStatus(
                                          u.id,
                                          u.status === "active" ? "disabled" : "active",
                                        ),
                                      children: [
                                        u.status === "active"
                                          ? _jsx(X, { className: "h-3.5 w-3.5 mr-2" })
                                          : _jsx(Check, { className: "h-3.5 w-3.5 mr-2" }),
                                        u.status === "active" ? "禁用账号" : "恢复账号",
                                      ],
                                    }),
                                    _jsxs(DropdownMenuItem, {
                                      className:
                                        "text-destructive focus:bg-destructive/10 focus:text-destructive",
                                      onSelect: async (e) => {
                                        e.preventDefault();
                                        const ok = await confirm({
                                          title: "删除确认",
                                          description: "确定要删除此用户？",
                                          confirmText: "删除",
                                          variant: "destructive",
                                        });
                                        if (ok) {
                                          try {
                                            await deleteUserMutation.mutateAsync(u.id);
                                            toast({ title: "已删除用户" });
                                          } catch (e) {
                                            toast({
                                              variant: "destructive",
                                              title: "删除失败",
                                              description: e.message,
                                            });
                                          }
                                        }
                                      },
                                      children: [
                                        _jsx(Trash2, { className: "h-3.5 w-3.5 mr-2" }),
                                        " \u5220\u9664\u7528\u6237",
                                      ],
                                    }),
                                  ],
                                }),
                              ],
                            }),
                          }),
                        ],
                      },
                      u.id,
                    ),
                  ),
            }),
          ],
        }),
      }),
    ],
  });
}
