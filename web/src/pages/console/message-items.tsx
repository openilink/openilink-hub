import { useState } from "react";
import { Play, FileText, Download, Volume2, AlertCircle, Loader2 } from "lucide-react";
import { MediaLightbox } from "./media-lightbox";

export interface MessageItemData {
  type: string;
  text?: string;
  file_name?: string;
}

interface MessageItemProps {
  item: MessageItemData;
  /** Index of this item within the message's item_list */
  index: number;
  /** Storage key map from message.media_keys, e.g. {"0": "botid/2024/01/01/xxx.jpg"} */
  mediaKeys?: Record<string, string>;
  mediaStatus?: string;
  direction: string;
}

function toMediaSrc(key: string): string {
  // If the key is already an absolute or root-relative URL, use as-is
  if (key.startsWith("http://") || key.startsWith("https://") || key.startsWith("/")) {
    return key;
  }
  return `/api/v1/media/${key}`;
}

function mediaUrl(mediaKeys: Record<string, string> | undefined, index: number): string | null {
  if (!mediaKeys) return null;
  const key = mediaKeys[String(index)];
  if (!key) return null;
  return toMediaSrc(key);
}

function thumbUrl(mediaKeys: Record<string, string> | undefined, index: number): string | null {
  if (!mediaKeys) return null;
  const key = mediaKeys[`${index}_thumb`];
  if (!key) return null;
  return toMediaSrc(key);
}

// --- Text ---
export function TextItem({ item }: { item: MessageItemData }) {
  if (!item.text) return null;
  return (
    <p className="leading-relaxed whitespace-pre-wrap break-words">{item.text}</p>
  );
}

// --- Image ---
export function ImageItem({ item, index, mediaKeys, mediaStatus }: MessageItemProps) {
  const [lightbox, setLightbox] = useState(false);
  const src = mediaUrl(mediaKeys, index);
  const thumb = thumbUrl(mediaKeys, index) || src;

  if (mediaStatus === "downloading") {
    return (
      <div className="flex items-center gap-2 text-xs text-muted-foreground py-2">
        <Loader2 className="h-4 w-4 animate-spin" />
        <span>图片下载中...</span>
      </div>
    );
  }

  if (!src) {
    return (
      <div className="flex items-center gap-2 text-xs text-muted-foreground py-2">
        <AlertCircle className="h-4 w-4" />
        <span>[图片]{item.file_name ? ` ${item.file_name}` : ""}</span>
      </div>
    );
  }

  return (
    <>
      {item.text && (
        <p className="leading-relaxed whitespace-pre-wrap break-words text-xs opacity-70 mb-1">{item.text}</p>
      )}
      <button
        type="button"
        onClick={() => setLightbox(true)}
        className="block rounded-lg focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary"
        aria-label={`查看图片${item.file_name ? ` ${item.file_name}` : ""}`}
      >
        <img
          src={thumb!}
          alt={item.file_name || "图片"}
          className="max-w-full max-h-64 rounded-lg hover:opacity-90 transition-opacity"
          loading="lazy"
        />
      </button>
      {lightbox && (
        <MediaLightbox
          type="image"
          src={src}
          alt={item.file_name}
          onClose={() => setLightbox(false)}
        />
      )}
    </>
  );
}

// --- Video ---
export function VideoItem({ item, index, mediaKeys, mediaStatus }: MessageItemProps) {
  const [lightbox, setLightbox] = useState(false);
  const src = mediaUrl(mediaKeys, index);
  const thumb = thumbUrl(mediaKeys, index);

  if (mediaStatus === "downloading") {
    return (
      <div className="flex items-center gap-2 text-xs text-muted-foreground py-2">
        <Loader2 className="h-4 w-4 animate-spin" />
        <span>视频下载中...</span>
      </div>
    );
  }

  if (!src) {
    return (
      <div className="flex items-center gap-2 text-xs text-muted-foreground py-2">
        <AlertCircle className="h-4 w-4" />
        <span>[视频]{item.file_name ? ` ${item.file_name}` : ""}</span>
      </div>
    );
  }

  return (
    <>
      {item.text && (
        <p className="leading-relaxed whitespace-pre-wrap break-words text-xs opacity-70 mb-1">{item.text}</p>
      )}
      <button
        type="button"
        onClick={() => setLightbox(true)}
        className="relative block max-w-full max-h-64 rounded-lg cursor-pointer group overflow-hidden focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary"
        aria-label={`播放视频${item.file_name ? ` ${item.file_name}` : ""}`}
      >
        {thumb ? (
          <img src={thumb} alt={item.file_name || "视频"} className="max-w-full max-h-64 rounded-lg" loading="lazy" />
        ) : (
          <div className="w-48 h-32 bg-muted rounded-lg flex items-center justify-center">
            <Play className="h-8 w-8 text-muted-foreground" />
          </div>
        )}
        <div className="absolute inset-0 flex items-center justify-center bg-black/20 group-hover:bg-black/30 transition-colors rounded-lg">
          <div className="h-12 w-12 rounded-full bg-white/90 flex items-center justify-center shadow-lg">
            <Play className="h-6 w-6 text-black ml-0.5" />
          </div>
        </div>
      </button>
      {lightbox && (
        <MediaLightbox
          type="video"
          src={src}
          onClose={() => setLightbox(false)}
        />
      )}
    </>
  );
}

// --- Voice ---
export function VoiceItem({ item, index, mediaKeys, mediaStatus }: MessageItemProps) {
  const src = mediaUrl(mediaKeys, index);

  if (mediaStatus === "downloading") {
    return (
      <div className="flex items-center gap-2 text-xs text-muted-foreground py-2">
        <Loader2 className="h-4 w-4 animate-spin" />
        <span>语音下载中...</span>
      </div>
    );
  }

  if (!src) {
    return (
      <div className="flex items-center gap-2 text-xs text-muted-foreground py-1">
        <Volume2 className="h-4 w-4" />
        <span>[语音消息]</span>
      </div>
    );
  }

  return (
    <div className="flex items-center gap-2 min-w-[180px]">
      <Volume2 className="h-4 w-4 shrink-0 text-muted-foreground" />
      <audio controls preload="none" className="h-8 max-w-[240px] w-full" aria-label="语音消息">
        <source src={src} />
      </audio>
    </div>
  );
}

// --- File ---
export function FileItem({ item, index, mediaKeys, mediaStatus, direction }: MessageItemProps) {
  const src = mediaUrl(mediaKeys, index);

  if (mediaStatus === "downloading") {
    return (
      <div className="flex items-center gap-2 text-xs text-muted-foreground py-2">
        <Loader2 className="h-4 w-4 animate-spin" />
        <span>文件下载中...</span>
      </div>
    );
  }

  return (
    <>
      {item.text && (
        <p className="leading-relaxed whitespace-pre-wrap break-words text-xs opacity-70 mb-1">{item.text}</p>
      )}
      <div className={`flex items-center gap-3 px-3 py-2.5 rounded-xl border ${direction === "inbound" ? "bg-muted/50 border-border/50" : "bg-white/10 border-white/20"}`}>
        <div className={`h-9 w-9 rounded-lg flex items-center justify-center shrink-0 ${direction === "inbound" ? "bg-primary/10 text-primary" : "bg-white/20 text-white"}`}>
          <FileText className="h-4 w-4" />
        </div>
        <div className="flex-1 min-w-0">
          <p className="text-sm font-medium truncate">{item.file_name || "文件"}</p>
        </div>
        {src && (
          <a
            href={src}
            download={item.file_name || true}
            onClick={(e) => e.stopPropagation()}
            className={`shrink-0 ${direction === "inbound" ? "text-muted-foreground hover:text-foreground" : "text-white/70 hover:text-white"}`}
            aria-label={`下载 ${item.file_name || "文件"}`}
          >
            <Download className="h-4 w-4" />
          </a>
        )}
      </div>
    </>
  );
}

// --- Dispatcher ---
export function MessageItem(props: MessageItemProps) {
  const { item } = props;
  switch (item.type) {
    case "text":
      return <TextItem item={item} />;
    case "image":
      return <ImageItem {...props} />;
    case "video":
      return <VideoItem {...props} />;
    case "voice":
      return <VoiceItem {...props} />;
    case "file":
      return <FileItem {...props} />;
    default:
      return <TextItem item={item} />;
  }
}
