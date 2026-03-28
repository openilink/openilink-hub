import { useState, useCallback, useRef } from "react";
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

  const isDestructive = (options.variant || "destructive") === "destructive";

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
