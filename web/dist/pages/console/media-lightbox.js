import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { useEffect, useCallback, useRef } from "react";
import { X } from "lucide-react";
export function MediaLightbox({ type, src, alt, onClose }) {
  const dialogRef = useRef(null);
  const previousFocusRef = useRef(null);
  const handleKeyDown = useCallback(
    (e) => {
      if (e.key === "Escape") onClose();
      // Simple focus trap: keep focus inside the dialog
      if (e.key === "Tab") {
        const focusable = dialogRef.current?.querySelectorAll(
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
    const closeBtn = dialogRef.current?.querySelector("button");
    closeBtn?.focus();
    return () => {
      document.removeEventListener("keydown", handleKeyDown);
      // Restore focus on close
      if (previousFocusRef.current instanceof HTMLElement) {
        previousFocusRef.current.focus();
      }
    };
  }, [handleKeyDown]);
  return _jsxs("div", {
    ref: dialogRef,
    role: "dialog",
    "aria-modal": "true",
    "aria-label":
      type === "image" ? `查看图片${alt ? ` ${alt}` : ""}` : `播放视频${alt ? ` ${alt}` : ""}`,
    className: "fixed inset-0 z-50 flex items-center justify-center bg-black/80 backdrop-blur-sm",
    onClick: onClose,
    children: [
      _jsx("button", {
        onClick: onClose,
        className:
          "absolute top-4 right-4 p-2 rounded-full bg-white/10 hover:bg-white/20 text-white transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-white",
        "aria-label": "\u5173\u95ED\u9884\u89C8",
        children: _jsx(X, { className: "h-6 w-6" }),
      }),
      _jsx("div", {
        className: "max-w-[90vw] max-h-[90vh] flex items-center justify-center",
        onClick: (e) => e.stopPropagation(),
        children:
          type === "image"
            ? _jsx("img", {
                src: src,
                alt: alt || "",
                className: "max-w-full max-h-[90vh] object-contain rounded-lg shadow-2xl",
              })
            : _jsx("video", {
                src: src,
                controls: true,
                autoPlay: true,
                className: "max-w-full max-h-[90vh] rounded-lg shadow-2xl",
              }),
      }),
    ],
  });
}
