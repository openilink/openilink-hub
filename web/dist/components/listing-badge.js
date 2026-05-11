import { jsx as _jsx } from "react/jsx-runtime";
import { Badge } from "@/components/ui/badge";
export function ListingBadge({ listing }) {
  if (listing === "listed")
    return _jsx(Badge, { variant: "default", children: "\u5DF2\u4E0A\u67B6" });
  if (listing === "pending")
    return _jsx(Badge, {
      variant: "outline",
      className: "text-orange-500 border-orange-500",
      children: "\u5BA1\u6838\u4E2D",
    });
  if (listing === "rejected")
    return _jsx(Badge, { variant: "destructive", children: "\u5DF2\u62D2\u7EDD" });
  return _jsx(Badge, { variant: "secondary", children: "\u672A\u4E0A\u67B6" });
}
