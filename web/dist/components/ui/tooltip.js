"use client";
import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { Provider, Root, Trigger, Portal, Content, Arrow } from "@radix-ui/react-tooltip";
import { cn } from "@/lib/utils";
function TooltipProvider({ delayDuration = 0, ...props }) {
  return _jsx(Provider, {
    "data-slot": "tooltip-provider",
    delayDuration: delayDuration,
    ...props,
  });
}
function Tooltip({ ...props }) {
  return _jsx(Root, { "data-slot": "tooltip", ...props });
}
function TooltipTrigger({ ...props }) {
  return _jsx(Trigger, { "data-slot": "tooltip-trigger", ...props });
}
function TooltipContent({ className, sideOffset = 0, children, ...props }) {
  return _jsx(Portal, {
    children: _jsxs(Content, {
      "data-slot": "tooltip-content",
      sideOffset: sideOffset,
      className: cn(
        "z-50 inline-flex w-fit max-w-xs origin-(--radix-tooltip-content-transform-origin) items-center gap-1.5 rounded-md bg-foreground px-3 py-1.5 text-xs text-background has-data-[slot=kbd]:pr-1.5 data-[side=bottom]:slide-in-from-top-2 data-[side=left]:slide-in-from-right-2 data-[side=right]:slide-in-from-left-2 data-[side=top]:slide-in-from-bottom-2 **:data-[slot=kbd]:relative **:data-[slot=kbd]:isolate **:data-[slot=kbd]:z-50 **:data-[slot=kbd]:rounded-sm data-[state=delayed-open]:animate-in data-[state=delayed-open]:fade-in-0 data-[state=delayed-open]:zoom-in-95 data-open:animate-in data-open:fade-in-0 data-open:zoom-in-95 data-closed:animate-out data-closed:fade-out-0 data-closed:zoom-out-95",
        className,
      ),
      ...props,
      children: [
        children,
        _jsx(Arrow, {
          className:
            "z-50 size-2.5 translate-y-[calc(-50%_-_2px)] rotate-45 rounded-[2px] bg-foreground fill-foreground",
        }),
      ],
    }),
  });
}
export { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger };
