import { useState, useCallback, useRef, type FormEvent } from "react";
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

interface ConfirmOptions {
  title?: string;
  description: string;
  confirmText?: string;
  cancelText?: string;
  variant?: "default" | "destructive";
}

type ResolveRef = ((value: boolean) => void) | null;

export function useConfirm() {
  const [open, setOpen] = useState(false);
  const [options, setOptions] = useState<ConfirmOptions>({
    description: "",
  });
  const resolveRef = useRef<ResolveRef>(null);

  const confirm = useCallback((opts: ConfirmOptions | string) => {
    // Resolve any pending confirmation as cancelled
    if (resolveRef.current) {
      resolveRef.current(false);
      resolveRef.current = null;
    }
    const o = typeof opts === "string" ? { description: opts } : opts;
    setOptions(o);
    setOpen(true);
    return new Promise<boolean>((resolve) => {
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

  const ConfirmDialog = (
    <AlertDialog open={open} onOpenChange={(v) => { if (!v) handleCancel(); }}>
      <AlertDialogContent className="max-w-sm">
        <AlertDialogHeader>
          <AlertDialogTitle className="text-base">{options.title || "确认操作"}</AlertDialogTitle>
          <AlertDialogDescription className="text-sm">{options.description}</AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter className="gap-2 sm:gap-0">
          <AlertDialogCancel
            className={cn(buttonVariants({ variant: "ghost", size: "sm" }), "mt-0 border-0")}
            onClick={handleCancel}
            autoFocus={isDestructive}
          >
            {options.cancelText || "取消"}
          </AlertDialogCancel>
          <AlertDialogAction
            className={cn(buttonVariants({ variant: isDestructive ? "destructive" : "default", size: "sm" }))}
            onClick={handleConfirm}
            autoFocus={!isDestructive}
          >
            {options.confirmText || "确认"}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );

  return { confirm, ConfirmDialog };
}

// --- usePrompt: text input variant ---

interface PromptOptions {
  title?: string;
  description: string;
  placeholder?: string;
  confirmText?: string;
  cancelText?: string;
}

type PromptResolveRef = ((value: string | null) => void) | null;

export function usePrompt() {
  const [open, setOpen] = useState(false);
  const [options, setOptions] = useState<PromptOptions>({ description: "" });
  const [inputValue, setInputValue] = useState("");
  const resolveRef = useRef<PromptResolveRef>(null);

  const prompt = useCallback((opts: PromptOptions | string) => {
    if (resolveRef.current) {
      resolveRef.current(null);
      resolveRef.current = null;
    }
    const o = typeof opts === "string" ? { description: opts } : opts;
    setOptions(o);
    setInputValue("");
    setOpen(true);
    return new Promise<string | null>((resolve) => {
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

  const handleSubmit = useCallback((e: FormEvent) => {
    e.preventDefault();
    if (!inputValue.trim()) return;
    handleConfirm();
  }, [handleConfirm, inputValue]);

  const PromptDialog = (
    <AlertDialog open={open} onOpenChange={(v: boolean) => { if (!v) handleCancel(); }}>
      <AlertDialogContent className="max-w-sm">
        <AlertDialogHeader>
          <AlertDialogTitle className="text-base">{options.title || "请输入"}</AlertDialogTitle>
          <AlertDialogDescription className="text-sm">{options.description}</AlertDialogDescription>
        </AlertDialogHeader>
        <form onSubmit={handleSubmit}>
          <Input
            value={inputValue}
            onChange={(e) => setInputValue(e.target.value)}
            placeholder={options.placeholder}
            className="mb-4"
            autoFocus
          />
          <AlertDialogFooter className="gap-2 sm:gap-0">
            <AlertDialogCancel
              type="button"
              className={cn(buttonVariants({ variant: "ghost", size: "sm" }), "mt-0 border-0")}
              onClick={handleCancel}
            >
              {options.cancelText || "取消"}
            </AlertDialogCancel>
            <AlertDialogAction
              type="submit"
              className={cn(buttonVariants({ size: "sm" }))}
              disabled={!inputValue.trim()}
            >
              {options.confirmText || "确认"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </form>
      </AlertDialogContent>
    </AlertDialog>
  );

  return { prompt, PromptDialog };
}
