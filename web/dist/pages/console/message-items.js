import { jsx as _jsx, jsxs as _jsxs, Fragment as _Fragment } from "react/jsx-runtime";
import { useState } from "react";
import { Play, FileText, Download, Volume2, AlertCircle, Loader2 } from "lucide-react";
import { MediaLightbox } from "./media-lightbox";
function toMediaSrc(key) {
  // If the key is already an absolute or root-relative URL, use as-is
  if (key.startsWith("http://") || key.startsWith("https://") || key.startsWith("/")) {
    return key;
  }
  return `/api/v1/media/${key}`;
}
function mediaUrl(mediaKeys, index) {
  if (!mediaKeys) return null;
  const key = mediaKeys[String(index)];
  if (!key) return null;
  return toMediaSrc(key);
}
function thumbUrl(mediaKeys, index) {
  if (!mediaKeys) return null;
  const key = mediaKeys[`${index}_thumb`];
  if (!key) return null;
  return toMediaSrc(key);
}
// --- Text ---
export function TextItem({ item }) {
  if (!item.text) return null;
  return _jsx("p", {
    className: "leading-relaxed whitespace-pre-wrap break-words",
    children: item.text,
  });
}
// Helper: render item.text as caption if present
function Caption({ text }) {
  if (!text) return null;
  return _jsx("p", {
    className: "leading-relaxed whitespace-pre-wrap break-words text-xs opacity-70 mb-1",
    children: text,
  });
}
// --- Image ---
export function ImageItem({ item, index, mediaKeys, mediaStatus }) {
  const [lightbox, setLightbox] = useState(false);
  const src = mediaUrl(mediaKeys, index);
  const thumb = thumbUrl(mediaKeys, index) || src;
  if (mediaStatus === "downloading") {
    return _jsxs(_Fragment, {
      children: [
        _jsx(Caption, { text: item.text }),
        _jsxs("div", {
          className: "flex items-center gap-2 text-xs text-muted-foreground py-2",
          children: [
            _jsx(Loader2, { className: "h-4 w-4 animate-spin" }),
            _jsx("span", { children: "\u56FE\u7247\u4E0B\u8F7D\u4E2D..." }),
          ],
        }),
      ],
    });
  }
  if (!src) {
    return _jsxs(_Fragment, {
      children: [
        _jsx(Caption, { text: item.text }),
        _jsxs("div", {
          className: "flex items-center gap-2 text-xs text-muted-foreground py-2",
          children: [
            _jsx(AlertCircle, { className: "h-4 w-4" }),
            _jsxs("span", {
              children: ["[\u56FE\u7247]", item.file_name ? ` ${item.file_name}` : ""],
            }),
          ],
        }),
      ],
    });
  }
  return _jsxs(_Fragment, {
    children: [
      _jsx(Caption, { text: item.text }),
      _jsx("button", {
        type: "button",
        onClick: () => setLightbox(true),
        className:
          "block rounded-lg focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary",
        "aria-label": `查看图片${item.file_name ? ` ${item.file_name}` : ""}`,
        children: _jsx("img", {
          src: thumb,
          alt: item.file_name || "图片",
          className: "max-w-full max-h-64 rounded-lg hover:opacity-90 transition-opacity",
          loading: "lazy",
        }),
      }),
      lightbox &&
        _jsx(MediaLightbox, {
          type: "image",
          src: src,
          alt: item.file_name,
          onClose: () => setLightbox(false),
        }),
    ],
  });
}
// --- Video ---
export function VideoItem({ item, index, mediaKeys, mediaStatus }) {
  const [lightbox, setLightbox] = useState(false);
  const src = mediaUrl(mediaKeys, index);
  const thumb = thumbUrl(mediaKeys, index);
  if (mediaStatus === "downloading") {
    return _jsxs(_Fragment, {
      children: [
        _jsx(Caption, { text: item.text }),
        _jsxs("div", {
          className: "flex items-center gap-2 text-xs text-muted-foreground py-2",
          children: [
            _jsx(Loader2, { className: "h-4 w-4 animate-spin" }),
            _jsx("span", { children: "\u89C6\u9891\u4E0B\u8F7D\u4E2D..." }),
          ],
        }),
      ],
    });
  }
  if (!src) {
    return _jsxs(_Fragment, {
      children: [
        _jsx(Caption, { text: item.text }),
        _jsxs("div", {
          className: "flex items-center gap-2 text-xs text-muted-foreground py-2",
          children: [
            _jsx(AlertCircle, { className: "h-4 w-4" }),
            _jsxs("span", {
              children: ["[\u89C6\u9891]", item.file_name ? ` ${item.file_name}` : ""],
            }),
          ],
        }),
      ],
    });
  }
  return _jsxs(_Fragment, {
    children: [
      _jsx(Caption, { text: item.text }),
      _jsxs("button", {
        type: "button",
        onClick: () => setLightbox(true),
        className:
          "relative block max-w-full max-h-64 rounded-lg cursor-pointer group overflow-hidden focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary",
        "aria-label": `播放视频${item.file_name ? ` ${item.file_name}` : ""}`,
        children: [
          thumb
            ? _jsx("img", {
                src: thumb,
                alt: item.file_name || "视频",
                className: "max-w-full max-h-64 rounded-lg",
                loading: "lazy",
              })
            : _jsx("div", {
                className: "w-48 h-32 bg-muted rounded-lg flex items-center justify-center",
                children: _jsx(Play, { className: "h-8 w-8 text-muted-foreground" }),
              }),
          _jsx("div", {
            className:
              "absolute inset-0 flex items-center justify-center bg-black/20 group-hover:bg-black/30 transition-colors rounded-lg",
            children: _jsx("div", {
              className:
                "h-12 w-12 rounded-full bg-white/90 flex items-center justify-center shadow-lg",
              children: _jsx(Play, { className: "h-6 w-6 text-black ml-0.5" }),
            }),
          }),
        ],
      }),
      lightbox &&
        _jsx(MediaLightbox, {
          type: "video",
          src: src,
          alt: item.file_name,
          onClose: () => setLightbox(false),
        }),
    ],
  });
}
// --- Voice ---
export function VoiceItem({ item, index, mediaKeys, mediaStatus }) {
  const src = mediaUrl(mediaKeys, index);
  if (mediaStatus === "downloading") {
    return _jsxs(_Fragment, {
      children: [
        _jsx(Caption, { text: item.text }),
        _jsxs("div", {
          className: "flex items-center gap-2 text-xs text-muted-foreground py-2",
          children: [
            _jsx(Loader2, { className: "h-4 w-4 animate-spin" }),
            _jsx("span", { children: "\u8BED\u97F3\u4E0B\u8F7D\u4E2D..." }),
          ],
        }),
      ],
    });
  }
  if (!src) {
    return _jsxs(_Fragment, {
      children: [
        _jsx(Caption, { text: item.text }),
        _jsxs("div", {
          className: "flex items-center gap-2 text-xs text-muted-foreground py-1",
          children: [
            _jsx(Volume2, { className: "h-4 w-4" }),
            _jsx("span", { children: "[\u8BED\u97F3\u6D88\u606F]" }),
          ],
        }),
      ],
    });
  }
  return _jsxs(_Fragment, {
    children: [
      _jsx(Caption, { text: item.text }),
      _jsxs("div", {
        className: "flex items-center gap-2 min-w-[180px]",
        children: [
          _jsx(Volume2, { className: "h-4 w-4 shrink-0 text-muted-foreground" }),
          _jsx("audio", {
            controls: true,
            preload: "none",
            className: "h-8 max-w-[240px] w-full",
            "aria-label": "\u8BED\u97F3\u6D88\u606F",
            children: _jsx("source", { src: src }),
          }),
        ],
      }),
    ],
  });
}
// --- File ---
export function FileItem({ item, index, mediaKeys, mediaStatus, direction }) {
  const src = mediaUrl(mediaKeys, index);
  if (mediaStatus === "downloading") {
    return _jsxs(_Fragment, {
      children: [
        _jsx(Caption, { text: item.text }),
        _jsxs("div", {
          className: "flex items-center gap-2 text-xs text-muted-foreground py-2",
          children: [
            _jsx(Loader2, { className: "h-4 w-4 animate-spin" }),
            _jsx("span", { children: "\u6587\u4EF6\u4E0B\u8F7D\u4E2D..." }),
          ],
        }),
      ],
    });
  }
  return _jsxs(_Fragment, {
    children: [
      _jsx(Caption, { text: item.text }),
      _jsxs("div", {
        className: `flex items-center gap-3 px-3 py-2.5 rounded-xl border ${direction === "inbound" ? "bg-muted/50 border-border/50" : "bg-white/10 border-white/20"}`,
        children: [
          _jsx("div", {
            className: `h-9 w-9 rounded-lg flex items-center justify-center shrink-0 ${direction === "inbound" ? "bg-primary/10 text-primary" : "bg-white/20 text-white"}`,
            children: _jsx(FileText, { className: "h-4 w-4" }),
          }),
          _jsx("div", {
            className: "flex-1 min-w-0",
            children: _jsx("p", {
              className: "text-sm font-medium truncate",
              children: item.file_name || "文件",
            }),
          }),
          src &&
            _jsx("a", {
              href: src,
              download: item.file_name || "",
              onClick: (e) => e.stopPropagation(),
              className: `shrink-0 ${direction === "inbound" ? "text-muted-foreground hover:text-foreground" : "text-white/70 hover:text-white"}`,
              "aria-label": `下载 ${item.file_name || "文件"}`,
              children: _jsx(Download, { className: "h-4 w-4" }),
            }),
        ],
      }),
    ],
  });
}
// --- Dispatcher ---
export function MessageItem(props) {
  const { item } = props;
  switch (item.type) {
    case "text":
      return _jsx(TextItem, { item: item });
    case "image":
      return _jsx(ImageItem, { ...props });
    case "video":
      return _jsx(VideoItem, { ...props });
    case "voice":
      return _jsx(VoiceItem, { ...props });
    case "file":
      return _jsx(FileItem, { ...props });
    default:
      return _jsx(TextItem, { item: item });
  }
}
