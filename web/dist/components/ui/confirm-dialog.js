import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { useState, useCallback, useRef } from "react";
import { Input } from "./input";
import {
  AlertDialog,
  AlertDialogContent,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogCancel,
  AlertDialogAction,
} from "./alert-dialog";
import { buttonVariants } from "./button";
import { cn } from "@/lib/utils";
export function useConfirm() {
  const [open, setOpen] = useState(false);
  const [options, setOptions] = useState({
    description: "",
  });
  const resolveRef = useRef(null);
  const confirm = useCallback((opts) => {
    // Resolve any pending confirmation as cancelled
    if (resolveRef.current) {
      resolveRef.current(false);
      resolveRef.current = null;
    }
    const o = typeof opts === "string" ? { description: opts } : opts;
    setOptions(o);
    setOpen(true);
    return new Promise((resolve) => {
      resolveRef.current = resolve;
    });
  }, []);
  const handleConfirm = useCallback(() => {
    setOpen(false);
    resolveRef.current?.(true);
    resolveRef.current = null;
  }, []);
  const handleCancel = useCallback(() => {
    setOpen(false);
    resolveRef.current?.(false);
    resolveRef.current = null;
  }, []);
  const isDestructive = options.variant === "destructive";
  const ConfirmDialog = _jsx(AlertDialog, {
    open: open,
    onOpenChange: (v) => {
      if (!v) handleCancel();
    },
    children: _jsxs(AlertDialogContent, {
      className: "max-w-sm",
      children: [
        _jsxs(AlertDialogHeader, {
          children: [
            _jsx(AlertDialogTitle, {
              className: "text-base",
              children: options.title || "确认操作",
            }),
            _jsx(AlertDialogDescription, { className: "text-sm", children: options.description }),
          ],
        }),
        _jsxs(AlertDialogFooter, {
          className: "gap-2 sm:gap-0",
          children: [
            _jsx(AlertDialogCancel, {
              className: cn(buttonVariants({ variant: "ghost", size: "sm" }), "mt-0 border-0"),
              onClick: handleCancel,
              autoFocus: isDestructive,
              children: options.cancelText || "取消",
            }),
            _jsx(AlertDialogAction, {
              className: cn(
                buttonVariants({ variant: isDestructive ? "destructive" : "default", size: "sm" }),
              ),
              onClick: handleConfirm,
              autoFocus: !isDestructive,
              children: options.confirmText || "确认",
            }),
          ],
        }),
      ],
    }),
  });
  return { confirm, ConfirmDialog };
}
export function usePrompt() {
  const [open, setOpen] = useState(false);
  const [options, setOptions] = useState({ description: "" });
  const [inputValue, setInputValue] = useState("");
  const resolveRef = useRef(null);
  const prompt = useCallback((opts) => {
    if (resolveRef.current) {
      resolveRef.current(null);
      resolveRef.current = null;
    }
    const o = typeof opts === "string" ? { description: opts } : opts;
    setOptions(o);
    setInputValue("");
    setOpen(true);
    return new Promise((resolve) => {
      resolveRef.current = resolve;
    });
  }, []);
  const handleConfirm = useCallback(() => {
    setOpen(false);
    const trimmed = inputValue.trim();
    resolveRef.current?.(trimmed || null);
    resolveRef.current = null;
  }, [inputValue]);
  const handleCancel = useCallback(() => {
    setOpen(false);
    resolveRef.current?.(null);
    resolveRef.current = null;
  }, []);
  const handleSubmit = useCallback(
    (e) => {
      e.preventDefault();
      if (!inputValue.trim()) return;
      handleConfirm();
    },
    [handleConfirm, inputValue],
  );
  const PromptDialog = _jsx(AlertDialog, {
    open: open,
    onOpenChange: (v) => {
      if (!v) handleCancel();
    },
    children: _jsxs(AlertDialogContent, {
      className: "max-w-sm",
      children: [
        _jsxs(AlertDialogHeader, {
          children: [
            _jsx(AlertDialogTitle, { className: "text-base", children: options.title || "请输入" }),
            _jsx(AlertDialogDescription, { className: "text-sm", children: options.description }),
          ],
        }),
        _jsxs("form", {
          onSubmit: handleSubmit,
          children: [
            _jsx(Input, {
              value: inputValue,
              onChange: (e) => setInputValue(e.target.value),
              placeholder: options.placeholder,
              className: "mb-4",
              autoFocus: true,
            }),
            _jsxs(AlertDialogFooter, {
              className: "gap-2 sm:gap-0",
              children: [
                _jsx(AlertDialogCancel, {
                  type: "button",
                  className: cn(buttonVariants({ variant: "ghost", size: "sm" }), "mt-0 border-0"),
                  onClick: handleCancel,
                  children: options.cancelText || "取消",
                }),
                _jsx(AlertDialogAction, {
                  type: "submit",
                  className: cn(buttonVariants({ size: "sm" })),
                  disabled: !inputValue.trim(),
                  children: options.confirmText || "确认",
                }),
              ],
            }),
          ],
        }),
      ],
    }),
  });
  return { prompt, PromptDialog };
}
