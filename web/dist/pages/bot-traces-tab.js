import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { useNavigate } from "react-router-dom";
import { Badge } from "../components/ui/badge";
import { Button } from "../components/ui/button";
import { useBotPush } from "../lib/ws";
import { useBotTraces } from "../hooks/use-bots";
import { RefreshCw, Activity, ChevronRight } from "lucide-react";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Skeleton } from "@/components/ui/skeleton";
import { durationMs, StatusIcon } from "@/lib/trace-utils";
export function BotTracesTab({ botId }) {
  const navigate = useNavigate();
  const { data: rootSpans = [], isLoading: loading, refetch } = useBotTraces(botId, 100);
  // Subscribe to push events — cache is auto-invalidated by PushProvider.
  useBotPush(botId);
  return _jsxs("div", {
    className: "space-y-4",
    children: [
      _jsxs("div", {
        className: "flex items-center justify-between",
        children: [
          _jsxs("div", {
            className: "flex items-center gap-2",
            children: [
              _jsx(Activity, { className: "w-4 h-4 text-primary" }),
              _jsx("h3", {
                className: "text-sm font-semibold",
                children: "\u6D88\u606F\u65E5\u5FD7",
              }),
            ],
          }),
          _jsxs(Button, {
            variant: "outline",
            size: "sm",
            onClick: () => refetch(),
            disabled: loading,
            className: "h-8",
            children: [
              _jsx(RefreshCw, { className: `w-3.5 h-3.5 mr-1.5 ${loading ? "animate-spin" : ""}` }),
              " \u5237\u65B0",
            ],
          }),
        ],
      }),
      _jsx("div", {
        className: "rounded-xl border bg-card/50 overflow-hidden shadow-sm",
        children: _jsxs(Table, {
          children: [
            _jsx(TableHeader, {
              className: "bg-muted/30",
              children: _jsxs(TableRow, {
                children: [
                  _jsx(TableHead, { className: "w-[100px]", children: "\u72B6\u6001" }),
                  _jsx(TableHead, { children: "\u53D1\u9001\u8005" }),
                  _jsx(TableHead, {
                    className: "hidden md:table-cell",
                    children: "\u6838\u5FC3\u4E8B\u4EF6",
                  }),
                  _jsx(TableHead, { className: "text-right", children: "Tokens" }),
                  _jsx(TableHead, { className: "text-right", children: "\u8017\u65F6" }),
                  _jsx(TableHead, { className: "text-right", children: "\u65F6\u95F4" }),
                  _jsx(TableHead, { className: "w-8" }),
                ],
              }),
            }),
            _jsx(TableBody, {
              children: loading
                ? Array.from({ length: 5 }).map((_, i) =>
                    _jsxs(
                      TableRow,
                      {
                        children: [
                          _jsx(TableCell, { children: _jsx(Skeleton, { className: "h-4 w-16" }) }),
                          _jsx(TableCell, { children: _jsx(Skeleton, { className: "h-4 w-24" }) }),
                          _jsx(TableCell, {
                            children: _jsx(Skeleton, { className: "h-4 w-full" }),
                          }),
                          _jsx(TableCell, {
                            children: _jsx(Skeleton, { className: "h-4 w-10 ml-auto" }),
                          }),
                          _jsx(TableCell, {
                            children: _jsx(Skeleton, { className: "h-4 w-12 ml-auto" }),
                          }),
                          _jsx(TableCell, {
                            children: _jsx(Skeleton, { className: "h-4 w-20 ml-auto" }),
                          }),
                          _jsx(TableCell, { children: _jsx(Skeleton, { className: "h-4 w-4" }) }),
                        ],
                      },
                      i,
                    ),
                  )
                : rootSpans.length === 0
                  ? _jsx(TableRow, {
                      children: _jsx(TableCell, {
                        colSpan: 7,
                        className: "h-32 text-center text-muted-foreground italic",
                        children: "\u6682\u65E0\u8BB0\u5F55",
                      }),
                    })
                  : rootSpans.map((root) => {
                      const dur = durationMs(root);
                      const sender = root.attributes?.["message.sender"] || "System";
                      const content = root.attributes?.["message.content"] || root.name;
                      return _jsxs(
                        TableRow,
                        {
                          className: "cursor-pointer focus-visible:bg-muted/50",
                          tabIndex: 0,
                          onClick: () =>
                            navigate(`/dashboard/accounts/${botId}/traces/${root.trace_id}`),
                          onKeyDown: (e) => {
                            if (e.key === "Enter" || e.key === " ") {
                              e.preventDefault();
                              navigate(`/dashboard/accounts/${botId}/traces/${root.trace_id}`);
                            }
                          },
                          children: [
                            _jsx(TableCell, {
                              children: _jsxs("div", {
                                className: "flex items-center gap-2",
                                children: [
                                  _jsx(StatusIcon, { code: root.status_code, size: "w-3.5 h-3.5" }),
                                  _jsx(Badge, {
                                    variant: "secondary",
                                    className: "text-[9px] h-4 leading-none uppercase",
                                    children: root.attributes?.["message.type"] || "执行",
                                  }),
                                ],
                              }),
                            }),
                            _jsx(TableCell, {
                              className: "font-mono text-xs max-w-[120px] truncate",
                              children: sender,
                            }),
                            _jsx(TableCell, {
                              className: "text-xs text-muted-foreground truncate max-w-[200px]",
                              title:
                                root.status_code === "error" && root.status_message
                                  ? root.status_message
                                  : undefined,
                              children:
                                root.status_code === "error" && root.status_message
                                  ? _jsx("span", {
                                      className: "text-destructive",
                                      children: root.status_message,
                                    })
                                  : content,
                            }),
                            _jsx(TableCell, {
                              className: "text-right font-mono text-[10px] text-muted-foreground",
                              children: root.attributes?.["ai.tokens.total"] || "\u2014",
                            }),
                            _jsx(TableCell, {
                              className: "text-right font-mono text-[10px] text-muted-foreground",
                              children: dur > 0 ? `${dur}ms` : "<1ms",
                            }),
                            _jsx(TableCell, {
                              className: "text-right text-[10px] text-muted-foreground",
                              children: new Date(root.start_time).toLocaleTimeString([], {
                                hour: "2-digit",
                                minute: "2-digit",
                                second: "2-digit",
                              }),
                            }),
                            _jsx(TableCell, {
                              className: "w-8 px-2",
                              children: _jsx(ChevronRight, {
                                className: "h-4 w-4 text-muted-foreground",
                              }),
                            }),
                          ],
                        },
                        root.id,
                      );
                    }),
            }),
          ],
        }),
      }),
    ],
  });
}
