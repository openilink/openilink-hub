import { useState, useCallback, useRef } from "react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from "./dialog";
import { Button } from "./button";

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

  const ConfirmDialog = (
    <Dialog open={open} onOpenChange={(v) => { if (!v) handleCancel(); }}>
      <DialogContent className="max-w-sm">
        <DialogHeader>
          <DialogTitle className="text-base">{options.title || "确认操作"}</DialogTitle>
          <DialogDescription className="text-sm">{options.description}</DialogDescription>
        </DialogHeader>
        <DialogFooter className="gap-2 sm:gap-0">
          <Button variant="ghost" size="sm" onClick={handleCancel}>
            {options.cancelText || "取消"}
          </Button>
          <Button
            variant={options.variant || "destructive"}
            size="sm"
            onClick={handleConfirm}
            autoFocus
          >
            {options.confirmText || "确认"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );

  return { confirm, ConfirmDialog };
}
