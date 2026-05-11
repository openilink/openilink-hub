import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { useParams, Link } from "react-router-dom";
import { ArrowLeft } from "lucide-react";
import { Button } from "@/components/ui/button";
import { BotTracesTab } from "./bot-traces-tab";
export function TracesPage() {
  const { id } = useParams();
  if (!id) return null;
  return _jsxs("div", {
    className: "flex flex-col gap-6",
    children: [
      _jsx("div", {
        className: "flex items-center gap-2",
        children: _jsx(Button, {
          variant: "ghost",
          size: "sm",
          className: "gap-1.5 text-muted-foreground hover:text-foreground -ml-2",
          asChild: true,
          children: _jsxs(Link, {
            to: `/dashboard/accounts/${id}`,
            children: [_jsx(ArrowLeft, { className: "h-4 w-4" }), "\u8FD4\u56DE\u8D26\u53F7"],
          }),
        }),
      }),
      _jsx("div", {
        className: "flex items-start justify-between gap-4",
        children: _jsxs("div", {
          children: [
            _jsx("h1", {
              className: "text-2xl font-bold tracking-tight",
              children: "\u6D88\u606F\u8FFD\u8E2A",
            }),
            _jsx("p", {
              className: "text-sm text-muted-foreground mt-0.5",
              children: "\u67E5\u770B\u6D88\u606F\u5904\u7406\u5168\u94FE\u8DEF\u65E5\u5FD7\u3002",
            }),
          ],
        }),
      }),
      _jsx(BotTracesTab, { botId: id }),
    ],
  });
}
