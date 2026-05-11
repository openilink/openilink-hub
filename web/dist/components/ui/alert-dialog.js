import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import * as React from "react";
import {
  Root,
  Portal,
  Overlay,
  Content,
  Title,
  Description,
  Action,
  Cancel,
} from "@radix-ui/react-alert-dialog";
import { cn } from "@/lib/utils";
import { buttonVariants } from "./button";
const AlertDialog = Root;
const AlertDialogPortal = Portal;
const AlertDialogOverlay = React.forwardRef(({ className, ...props }, ref) =>
  _jsx(Overlay, {
    ref: ref,
    className: cn(
      "fixed inset-0 z-50 bg-black/80 data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0",
      className,
    ),
    ...props,
  }),
);
AlertDialogOverlay.displayName = Overlay.displayName;
const AlertDialogContent = React.forwardRef(({ className, ...props }, ref) =>
  _jsxs(AlertDialogPortal, {
    children: [
      _jsx(AlertDialogOverlay, {}),
      _jsx(Content, {
        ref: ref,
        className: cn(
          "fixed left-[50%] top-[50%] z-50 grid w-full max-w-lg translate-x-[-50%] translate-y-[-50%] gap-4 border bg-background p-6 shadow-lg duration-200 data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0 data-[state=closed]:zoom-out-95 data-[state=open]:zoom-in-95 data-[state=closed]:slide-out-to-left-1/2 data-[state=closed]:slide-out-to-top-[48%] data-[state=open]:slide-in-from-left-1/2 data-[state=open]:slide-in-from-top-[48%] sm:rounded-lg",
          className,
        ),
        ...props,
      }),
    ],
  }),
);
AlertDialogContent.displayName = Content.displayName;
const AlertDialogHeader = ({ className, ...props }) =>
  _jsx("div", {
    className: cn("flex flex-col space-y-2 text-center sm:text-left", className),
    ...props,
  });
AlertDialogHeader.displayName = "AlertDialogHeader";
const AlertDialogFooter = ({ className, ...props }) =>
  _jsx("div", {
    className: cn("flex flex-col-reverse sm:flex-row sm:justify-end sm:space-x-2", className),
    ...props,
  });
AlertDialogFooter.displayName = "AlertDialogFooter";
const AlertDialogTitle = React.forwardRef(({ className, ...props }, ref) =>
  _jsx(Title, { ref: ref, className: cn("text-lg font-semibold", className), ...props }),
);
AlertDialogTitle.displayName = Title.displayName;
const AlertDialogDescription = React.forwardRef(({ className, ...props }, ref) =>
  _jsx(Description, {
    ref: ref,
    className: cn("text-sm text-muted-foreground", className),
    ...props,
  }),
);
AlertDialogDescription.displayName = Description.displayName;
const AlertDialogAction = React.forwardRef(({ className, ...props }, ref) =>
  _jsx(Action, { ref: ref, className: cn(buttonVariants(), className), ...props }),
);
AlertDialogAction.displayName = Action.displayName;
const AlertDialogCancel = React.forwardRef(({ className, ...props }, ref) =>
  _jsx(Cancel, {
    ref: ref,
    className: cn(buttonVariants({ variant: "outline" }), "mt-2 sm:mt-0", className),
    ...props,
  }),
);
AlertDialogCancel.displayName = Cancel.displayName;
export {
  AlertDialog,
  AlertDialogPortal,
  AlertDialogOverlay,
  AlertDialogContent,
  AlertDialogHeader,
  AlertDialogFooter,
  AlertDialogTitle,
  AlertDialogDescription,
  AlertDialogAction,
  AlertDialogCancel,
};
