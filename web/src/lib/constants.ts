export const SCOPE_DESCRIPTIONS: Record<string, string> = {
  "messages.send": "在已授权的对话中发送消息",
  "messages.read": "查看消息和对话内容",
  "contacts.read": "查看联系人基本信息",
  "bot.read": "查看账号基本信息和状态",
};

export const SCOPES = [
  { key: "messages.send", label: "发送消息", description: "在已授权的对话中发送消息", category: "write" as const },
  { key: "messages.read", label: "读取消息", description: "查看消息和对话内容", category: "read" as const },
  { key: "contacts.read", label: "读取联系人", description: "查看联系人基本信息", category: "read" as const },
  { key: "bot.read", label: "读取账号信息", description: "查看账号基本信息和状态", category: "read" as const },
];

export const EVENT_TYPES = [
  { key: "message.text", label: "文本消息", description: "接收文本消息事件" },
  { key: "message.image", label: "图片消息", description: "接收图片消息事件" },
  { key: "message.video", label: "视频消息", description: "接收视频消息事件" },
  { key: "message.voice", label: "语音消息", description: "接收语音消息事件" },
  { key: "message.file", label: "文件消息", description: "接收文件消息事件" },
  { key: "message.location", label: "位置消息", description: "接收位置消息事件" },
  { key: "contact.added", label: "新增联系人", description: "新联系人添加事件" },
  { key: "group.join", label: "入群", description: "有人加入群组事件" },
  { key: "group.leave", label: "退群", description: "有人退出群组事件" },
];
