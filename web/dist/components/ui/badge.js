import { jsx as _jsx } from "react/jsx-runtime";
import { cn } from "../../lib/utils";
const variants = {
  default: "bg-primary text-primary-foreground shadow-sm hover:bg-primary/80",
  secondary: "bg-secondary text-secondary-foreground hover:bg-secondary/80",
  destructive: "bg-destructive text-destructive-foreground shadow-sm hover:bg-destructive/80",
  outline: "border text-foreground",
};
export function Badge({ variant = "default", className, children }) {
  return _jsx("span", {
    className: cn(
      "inline-flex items-center rounded-md border border-transparent px-2.5 py-0.5 text-xs font-medium transition-colors",
      variants[variant],
      className,
    ),
    children: children,
  });
}
