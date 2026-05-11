import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import {
  Root,
  Trigger,
  Close,
  Portal,
  Overlay,
  Content,
  Title,
  Description,
} from "@radix-ui/react-dialog";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { XIcon } from "lucide-react";
function Sheet({ ...props }) {
  return _jsx(Root, { "data-slot": "sheet", ...props });
}
function SheetTrigger({ ...props }) {
  return _jsx(Trigger, { "data-slot": "sheet-trigger", ...props });
}
function SheetClose({ ...props }) {
  return _jsx(Close, { "data-slot": "sheet-close", ...props });
}
function SheetPortal({ ...props }) {
  return _jsx(Portal, { "data-slot": "sheet-portal", ...props });
}
function SheetOverlay({ className, ...props }) {
  return _jsx(Overlay, {
    "data-slot": "sheet-overlay",
    className: cn(
      "fixed inset-0 z-50 bg-black/10 duration-100 supports-backdrop-filter:backdrop-blur-xs data-open:animate-in data-open:fade-in-0 data-closed:animate-out data-closed:fade-out-0",
      className,
    ),
    ...props,
  });
}
function SheetContent({ className, children, side = "right", showCloseButton = true, ...props }) {
  return _jsxs(SheetPortal, {
    children: [
      _jsx(SheetOverlay, {}),
      _jsxs(Content, {
        "data-slot": "sheet-content",
        "data-side": side,
        className: cn(
          "fixed z-50 flex flex-col gap-4 bg-background bg-clip-padding text-sm shadow-lg transition duration-200 ease-in-out data-[side=bottom]:inset-x-0 data-[side=bottom]:bottom-0 data-[side=bottom]:h-auto data-[side=bottom]:border-t data-[side=left]:inset-y-0 data-[side=left]:left-0 data-[side=left]:h-full data-[side=left]:w-3/4 data-[side=left]:border-r data-[side=right]:inset-y-0 data-[side=right]:right-0 data-[side=right]:h-full data-[side=right]:w-3/4 data-[side=right]:border-l data-[side=top]:inset-x-0 data-[side=top]:top-0 data-[side=top]:h-auto data-[side=top]:border-b data-[side=left]:sm:max-w-sm data-[side=right]:sm:max-w-sm data-open:animate-in data-open:fade-in-0 data-[side=bottom]:data-open:slide-in-from-bottom-10 data-[side=left]:data-open:slide-in-from-left-10 data-[side=right]:data-open:slide-in-from-right-10 data-[side=top]:data-open:slide-in-from-top-10 data-closed:animate-out data-closed:fade-out-0 data-[side=bottom]:data-closed:slide-out-to-bottom-10 data-[side=left]:data-closed:slide-out-to-left-10 data-[side=right]:data-closed:slide-out-to-right-10 data-[side=top]:data-closed:slide-out-to-top-10",
          className,
        ),
        ...props,
        children: [
          children,
          showCloseButton &&
            _jsx(Close, {
              "data-slot": "sheet-close",
              asChild: true,
              children: _jsxs(Button, {
                variant: "ghost",
                className: "absolute top-3 right-3",
                size: "icon-sm",
                children: [
                  _jsx(XIcon, {}),
                  _jsx("span", { className: "sr-only", children: "Close" }),
                ],
              }),
            }),
        ],
      }),
    ],
  });
}
function SheetHeader({ className, ...props }) {
  return _jsx("div", {
    "data-slot": "sheet-header",
    className: cn("flex flex-col gap-0.5 p-4", className),
    ...props,
  });
}
function SheetFooter({ className, ...props }) {
  return _jsx("div", {
    "data-slot": "sheet-footer",
    className: cn("mt-auto flex flex-col gap-2 p-4", className),
    ...props,
  });
}
function SheetTitle({ className, ...props }) {
  return _jsx(Title, {
    "data-slot": "sheet-title",
    className: cn("font-heading text-base font-medium text-foreground", className),
    ...props,
  });
}
function SheetDescription({ className, ...props }) {
  return _jsx(Description, {
    "data-slot": "sheet-description",
    className: cn("text-sm text-muted-foreground", className),
    ...props,
  });
}
export {
  Sheet,
  SheetTrigger,
  SheetClose,
  SheetContent,
  SheetHeader,
  SheetFooter,
  SheetTitle,
  SheetDescription,
};
