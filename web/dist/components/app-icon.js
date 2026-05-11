import { jsx as _jsx } from "react/jsx-runtime";
import { Blocks } from "lucide-react";
export function AppIcon({ icon, iconUrl, size = "h-12 w-12" }) {
  if (iconUrl)
    return _jsx("img", {
      src: iconUrl,
      alt: "",
      className: `${size} rounded-xl object-cover border`,
    });
  if (icon)
    return _jsx("div", {
      className: `${size} rounded-xl bg-muted flex items-center justify-center text-lg border`,
      children: icon,
    });
  return _jsx("div", {
    className: `${size} rounded-xl bg-muted flex items-center justify-center border`,
    children: _jsx(Blocks, { className: "h-5 w-5 text-muted-foreground/40" }),
  });
}
