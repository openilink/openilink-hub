import { useEffect, useCallback, useRef } from "react";
import { X } from "lucide-react";

interface MediaLightboxProps {
  type: "image" | "video";
  src: string;
  alt?: string;
  onClose: () => void;
}

export function MediaLightbox({ type, src, alt, onClose }: MediaLightboxProps) {
  const dialogRef = useRef<HTMLDivElement>(null);
  const previousFocusRef = useRef<Element | null>(null);

  const handleKeyDown = useCallback(
    (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
      // Simple focus trap: keep focus inside the dialog
      if (e.key === "Tab") {
        const focusable = dialogRef.current?.querySelectorAll<HTMLElement>(
          'button, [href], input, select, textarea, video[controls], audio[controls], [tabindex]:not([tabindex="-1"])',
        );
        if (!focusable?.length) return;
        const first = focusable[0];
        const last = focusable[focusable.length - 1];
        if (e.shiftKey && document.activeElement === first) {
          e.preventDefault();
          last.focus();
        } else if (!e.shiftKey && document.activeElement === last) {
          e.preventDefault();
          first.focus();
        }
      }
    },
    [onClose],
  );

  useEffect(() => {
    previousFocusRef.current = document.activeElement;
    document.addEventListener("keydown", handleKeyDown);
    // Focus the close button on open
    const closeBtn = dialogRef.current?.querySelector<HTMLElement>("button");
    closeBtn?.focus();
    return () => {
      document.removeEventListener("keydown", handleKeyDown);
      // Restore focus on close
      if (previousFocusRef.current instanceof HTMLElement) {
        previousFocusRef.current.focus();
      }
    };
  }, [handleKeyDown]);

  return (
    <div
      ref={dialogRef}
      role="dialog"
      aria-modal="true"
      aria-label={type === "image" ? `查看图片${alt ? ` ${alt}` : ""}` : `播放视频${alt ? ` ${alt}` : ""}`}
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/80 backdrop-blur-sm"
      onClick={onClose}
    >
      <button
        onClick={onClose}
        className="absolute top-4 right-4 p-2 rounded-full bg-white/10 hover:bg-white/20 text-white transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-white"
        aria-label="关闭预览"
      >
        <X className="h-6 w-6" />
      </button>
      <div
        className="max-w-[90vw] max-h-[90vh] flex items-center justify-center"
        onClick={(e) => e.stopPropagation()}
      >
        {type === "image" ? (
          <img
            src={src}
            alt={alt || ""}
            className="max-w-full max-h-[90vh] object-contain rounded-lg shadow-2xl"
          />
        ) : (
          <video
            src={src}
            controls
            autoPlay
            className="max-w-full max-h-[90vh] rounded-lg shadow-2xl"
          />
        )}
      </div>
    </div>
  );
}
