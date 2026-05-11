import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import {
  Root,
  Portal,
  Trigger,
  Content,
  Group,
  Item,
  CheckboxItem,
  ItemIndicator,
  RadioGroup,
  RadioItem,
  Label,
  Separator,
  Sub,
  SubTrigger,
  SubContent,
} from "@radix-ui/react-dropdown-menu";
import { cn } from "@/lib/utils";
import { CheckIcon, ChevronRightIcon } from "lucide-react";
function DropdownMenu({ ...props }) {
  return _jsx(Root, { "data-slot": "dropdown-menu", ...props });
}
function DropdownMenuPortal({ ...props }) {
  return _jsx(Portal, { "data-slot": "dropdown-menu-portal", ...props });
}
function DropdownMenuTrigger({ ...props }) {
  return _jsx(Trigger, { "data-slot": "dropdown-menu-trigger", ...props });
}
function DropdownMenuContent({ className, align = "start", sideOffset = 4, ...props }) {
  return _jsx(Portal, {
    children: _jsx(Content, {
      "data-slot": "dropdown-menu-content",
      sideOffset: sideOffset,
      align: align,
      className: cn(
        "z-50 max-h-(--radix-dropdown-menu-content-available-height) w-(--radix-dropdown-menu-trigger-width) min-w-32 origin-(--radix-dropdown-menu-content-transform-origin) overflow-x-hidden overflow-y-auto rounded-lg bg-popover p-1 text-popover-foreground shadow-md ring-1 ring-foreground/10 duration-100 data-[side=bottom]:slide-in-from-top-2 data-[side=left]:slide-in-from-right-2 data-[side=right]:slide-in-from-left-2 data-[side=top]:slide-in-from-bottom-2 data-[state=closed]:overflow-hidden data-open:animate-in data-open:fade-in-0 data-open:zoom-in-95 data-closed:animate-out data-closed:fade-out-0 data-closed:zoom-out-95",
        className,
      ),
      ...props,
    }),
  });
}
function DropdownMenuGroup({ ...props }) {
  return _jsx(Group, { "data-slot": "dropdown-menu-group", ...props });
}
function DropdownMenuItem({ className, inset, variant = "default", ...props }) {
  return _jsx(Item, {
    "data-slot": "dropdown-menu-item",
    "data-inset": inset,
    "data-variant": variant,
    className: cn(
      "group/dropdown-menu-item relative flex cursor-default items-center gap-1.5 rounded-md px-1.5 py-1 text-sm outline-hidden select-none focus:bg-accent focus:text-accent-foreground not-data-[variant=destructive]:focus:**:text-accent-foreground data-inset:pl-7 data-[variant=destructive]:text-destructive data-[variant=destructive]:focus:bg-destructive/10 data-[variant=destructive]:focus:text-destructive dark:data-[variant=destructive]:focus:bg-destructive/20 data-disabled:pointer-events-none data-disabled:opacity-50 [&_svg]:pointer-events-none [&_svg]:shrink-0 [&_svg:not([class*='size-'])]:size-4 data-[variant=destructive]:*:[svg]:text-destructive",
      className,
    ),
    ...props,
  });
}
function DropdownMenuCheckboxItem({ className, children, checked, inset, ...props }) {
  return _jsxs(CheckboxItem, {
    "data-slot": "dropdown-menu-checkbox-item",
    "data-inset": inset,
    className: cn(
      "relative flex cursor-default items-center gap-1.5 rounded-md py-1 pr-8 pl-1.5 text-sm outline-hidden select-none focus:bg-accent focus:text-accent-foreground focus:**:text-accent-foreground data-inset:pl-7 data-disabled:pointer-events-none data-disabled:opacity-50 [&_svg]:pointer-events-none [&_svg]:shrink-0 [&_svg:not([class*='size-'])]:size-4",
      className,
    ),
    checked: checked,
    ...props,
    children: [
      _jsx("span", {
        className: "pointer-events-none absolute right-2 flex items-center justify-center",
        "data-slot": "dropdown-menu-checkbox-item-indicator",
        children: _jsx(ItemIndicator, { children: _jsx(CheckIcon, {}) }),
      }),
      children,
    ],
  });
}
function DropdownMenuRadioGroup({ ...props }) {
  return _jsx(RadioGroup, { "data-slot": "dropdown-menu-radio-group", ...props });
}
function DropdownMenuRadioItem({ className, children, inset, ...props }) {
  return _jsxs(RadioItem, {
    "data-slot": "dropdown-menu-radio-item",
    "data-inset": inset,
    className: cn(
      "relative flex cursor-default items-center gap-1.5 rounded-md py-1 pr-8 pl-1.5 text-sm outline-hidden select-none focus:bg-accent focus:text-accent-foreground focus:**:text-accent-foreground data-inset:pl-7 data-disabled:pointer-events-none data-disabled:opacity-50 [&_svg]:pointer-events-none [&_svg]:shrink-0 [&_svg:not([class*='size-'])]:size-4",
      className,
    ),
    ...props,
    children: [
      _jsx("span", {
        className: "pointer-events-none absolute right-2 flex items-center justify-center",
        "data-slot": "dropdown-menu-radio-item-indicator",
        children: _jsx(ItemIndicator, { children: _jsx(CheckIcon, {}) }),
      }),
      children,
    ],
  });
}
function DropdownMenuLabel({ className, inset, ...props }) {
  return _jsx(Label, {
    "data-slot": "dropdown-menu-label",
    "data-inset": inset,
    className: cn(
      "px-1.5 py-1 text-xs font-medium text-muted-foreground data-inset:pl-7",
      className,
    ),
    ...props,
  });
}
function DropdownMenuSeparator({ className, ...props }) {
  return _jsx(Separator, {
    "data-slot": "dropdown-menu-separator",
    className: cn("-mx-1 my-1 h-px bg-border", className),
    ...props,
  });
}
function DropdownMenuShortcut({ className, ...props }) {
  return _jsx("span", {
    "data-slot": "dropdown-menu-shortcut",
    className: cn(
      "ml-auto text-xs tracking-widest text-muted-foreground group-focus/dropdown-menu-item:text-accent-foreground",
      className,
    ),
    ...props,
  });
}
function DropdownMenuSub({ ...props }) {
  return _jsx(Sub, { "data-slot": "dropdown-menu-sub", ...props });
}
function DropdownMenuSubTrigger({ className, inset, children, ...props }) {
  return _jsxs(SubTrigger, {
    "data-slot": "dropdown-menu-sub-trigger",
    "data-inset": inset,
    className: cn(
      "flex cursor-default items-center gap-1.5 rounded-md px-1.5 py-1 text-sm outline-hidden select-none focus:bg-accent focus:text-accent-foreground not-data-[variant=destructive]:focus:**:text-accent-foreground data-inset:pl-7 data-open:bg-accent data-open:text-accent-foreground [&_svg]:pointer-events-none [&_svg]:shrink-0 [&_svg:not([class*='size-'])]:size-4",
      className,
    ),
    ...props,
    children: [children, _jsx(ChevronRightIcon, { className: "ml-auto" })],
  });
}
function DropdownMenuSubContent({ className, ...props }) {
  return _jsx(SubContent, {
    "data-slot": "dropdown-menu-sub-content",
    className: cn(
      "z-50 min-w-[96px] origin-(--radix-dropdown-menu-content-transform-origin) overflow-hidden rounded-lg bg-popover p-1 text-popover-foreground shadow-lg ring-1 ring-foreground/10 duration-100 data-[side=bottom]:slide-in-from-top-2 data-[side=left]:slide-in-from-right-2 data-[side=right]:slide-in-from-left-2 data-[side=top]:slide-in-from-bottom-2 data-open:animate-in data-open:fade-in-0 data-open:zoom-in-95 data-closed:animate-out data-closed:fade-out-0 data-closed:zoom-out-95",
      className,
    ),
    ...props,
  });
}
export {
  DropdownMenu,
  DropdownMenuPortal,
  DropdownMenuTrigger,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuLabel,
  DropdownMenuItem,
  DropdownMenuCheckboxItem,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuSeparator,
  DropdownMenuShortcut,
  DropdownMenuSub,
  DropdownMenuSubTrigger,
  DropdownMenuSubContent,
};
