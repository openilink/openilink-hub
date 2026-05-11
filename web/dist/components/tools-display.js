import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { Badge } from "./ui/badge";
export function ToolsDisplay({ tools }) {
  if (!tools || tools.length === 0) return null;
  return _jsxs("div", {
    className: "space-y-2",
    children: [
      _jsx("h4", {
        className: "text-xs font-bold uppercase tracking-wider text-muted-foreground",
        children: "\u547D\u4EE4",
      }),
      _jsx("div", {
        className: "grid gap-2",
        children: tools.map((tool) =>
          _jsxs(
            "div",
            {
              className: "flex items-start gap-3 p-2.5 rounded-lg border bg-muted/20",
              children: [
                tool.command &&
                  _jsxs(Badge, {
                    variant: "secondary",
                    className: "font-mono text-xs shrink-0",
                    children: ["/", tool.command],
                  }),
                _jsxs("div", {
                  className: "min-w-0",
                  children: [
                    _jsx("p", {
                      className: "text-sm font-medium",
                      children: tool.description || tool.name,
                    }),
                    tool.parameters?.properties &&
                      _jsx("div", {
                        className: "flex flex-wrap gap-1 mt-1",
                        children: Object.entries(tool.parameters.properties).map(([key, prop]) =>
                          _jsxs(
                            "span",
                            {
                              className: "text-[10px] text-muted-foreground font-mono",
                              children: [
                                key,
                                prop.description &&
                                  _jsxs("span", {
                                    className: "text-muted-foreground/60",
                                    children: [" (", prop.description, ")"],
                                  }),
                              ],
                            },
                            key,
                          ),
                        ),
                      }),
                  ],
                }),
              ],
            },
            tool.name,
          ),
        ),
      }),
    ],
  });
}
/** Parse tools from API data (may be JSON string or array). */
export function parseTools(raw) {
  if (!raw) return [];
  if (Array.isArray(raw)) return raw;
  if (typeof raw === "string") {
    try {
      const parsed = JSON.parse(raw);
      return Array.isArray(parsed) ? parsed : [];
    } catch {
      return [];
    }
  }
  return [];
}
