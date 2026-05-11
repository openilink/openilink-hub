import { jsx as _jsx, Fragment as _Fragment } from "react/jsx-runtime";
// @vitest-environment jsdom
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { BotDetailPage } from "./bot-detail";
const navigateMock = vi.fn();
const confirmMock = vi.fn();
const deleteBotMock = vi.fn();
const toastMock = vi.fn();
vi.mock("react-router-dom", () => ({
  Link: ({ children, to, ...props }) =>
    _jsx("a", { href: typeof to === "string" ? to : "#", ...props, children: children }),
  useNavigate: () => navigateMock,
  useParams: () => ({ id: "bot-1" }),
}));
vi.mock("../lib/api", () => ({
  botDisplayName: (bot) => bot.display_name || bot.name,
}));
vi.mock("@/hooks/use-bots", () => ({
  useBot: () => ({
    data: {
      id: "bot-1",
      name: "bot-1",
      display_name: "Existing Bot",
      provider: "ilink",
      status: "connected",
      can_send: true,
      ai_enabled: false,
      ai_model: "",
      reminder_hours: 0,
    },
    isLoading: false,
  }),
  useBotApps: () => ({ data: [] }),
  useDeleteBot: () => ({
    mutate: (id, options) => {
      Promise.resolve(deleteBotMock(id))
        .then(() => options?.onSuccess?.())
        .catch((error) => options?.onError?.(error))
        .finally(() => options?.onSettled?.());
    },
  }),
  useUpdateBot: () => ({ mutate: vi.fn() }),
  useSetBotAI: () => ({ mutate: vi.fn() }),
  useSetBotAIModel: () => ({ mutate: vi.fn() }),
}));
vi.mock("@/hooks/use-apps", () => ({
  useApps: () => ({ data: [] }),
  useAvailableModels: () => ({ data: [] }),
}));
vi.mock("@/hooks/use-marketplace", () => ({
  useBuiltinApps: () => ({ data: [] }),
  useMarketplaceApps: () => ({ data: [] }),
  useSyncMarketplaceApp: () => ({ mutate: vi.fn() }),
}));
vi.mock("@/hooks/use-toast", () => ({
  useToast: () => ({ toast: toastMock }),
}));
vi.mock("@/components/ui/confirm-dialog", () => ({
  useConfirm: () => ({
    confirm: (...args) => confirmMock(...args),
    ConfirmDialog: null,
  }),
}));
vi.mock("@/components/ui/tooltip", () => ({
  Tooltip: ({ children }) => _jsx(_Fragment, { children: children }),
  TooltipTrigger: ({ children }) => _jsx(_Fragment, { children: children }),
  TooltipContent: ({ children }) => _jsx(_Fragment, { children: children }),
}));
describe("BotDetailPage", () => {
  let container;
  let root;
  beforeEach(() => {
    globalThis.IS_REACT_ACT_ENVIRONMENT = true;
    container = document.createElement("div");
    document.body.appendChild(container);
    root = createRoot(container);
    confirmMock.mockResolvedValue(true);
    deleteBotMock.mockResolvedValue({ ok: true });
  });
  afterEach(async () => {
    await act(async () => {
      root.unmount();
    });
    container.remove();
    vi.clearAllMocks();
  });
  async function renderPage() {
    await act(async () => {
      root.render(_jsx(BotDetailPage, {}));
    });
  }
  function getDeleteButton() {
    const deleteButton = container.querySelector('button[aria-label="删除账号"]');
    expect(deleteButton).not.toBeNull();
    return deleteButton;
  }
  async function clickDeleteButton() {
    const deleteButton = getDeleteButton();
    await act(async () => {
      deleteButton?.dispatchEvent(new MouseEvent("click", { bubbles: true }));
    });
  }
  it("deletes the current bot after confirmation", async () => {
    await renderPage();
    await clickDeleteButton();
    await vi.waitFor(() => {
      expect(confirmMock).toHaveBeenCalledTimes(1);
      expect(deleteBotMock).toHaveBeenCalledWith("bot-1");
      expect(navigateMock).toHaveBeenCalledWith("/dashboard/accounts");
      expect(toastMock).toHaveBeenCalled();
    });
  });
  it("does not delete when confirmation is cancelled", async () => {
    confirmMock.mockResolvedValue(false);
    await renderPage();
    await clickDeleteButton();
    await vi.waitFor(() => {
      expect(confirmMock).toHaveBeenCalledTimes(1);
    });
    expect(deleteBotMock).not.toHaveBeenCalled();
    expect(navigateMock).not.toHaveBeenCalled();
  });
  it("does not open duplicate confirmation dialogs while confirmation is pending", async () => {
    let resolveConfirm;
    confirmMock.mockImplementation(
      () =>
        new Promise((resolve) => {
          resolveConfirm = resolve;
        }),
    );
    await renderPage();
    const deleteButton = getDeleteButton();
    await act(async () => {
      deleteButton.click();
      deleteButton.click();
    });
    await vi.waitFor(() => {
      expect(confirmMock).toHaveBeenCalledTimes(1);
      expect(deleteButton.disabled).toBe(true);
    });
    resolveConfirm?.(false);
    await vi.waitFor(() => {
      expect(deleteButton.disabled).toBe(false);
    });
    expect(deleteBotMock).not.toHaveBeenCalled();
  });
  it("shows an error toast when deletion fails", async () => {
    deleteBotMock.mockRejectedValue(new Error("delete failed"));
    await renderPage();
    await clickDeleteButton();
    await vi.waitFor(() => {
      expect(confirmMock).toHaveBeenCalledTimes(1);
      expect(deleteBotMock).toHaveBeenCalledWith("bot-1");
      expect(toastMock).toHaveBeenCalledWith(
        expect.objectContaining({
          variant: "destructive",
          title: "删除失败",
          description: "delete failed",
        }),
      );
    });
    expect(navigateMock).not.toHaveBeenCalled();
  });
  it("does not trigger duplicate deletes while a delete is already in flight", async () => {
    let resolveDelete;
    deleteBotMock.mockImplementation(
      () =>
        new Promise((resolve) => {
          resolveDelete = resolve;
        }),
    );
    await renderPage();
    const deleteButton = getDeleteButton();
    await act(async () => {
      deleteButton.click();
      deleteButton.click();
    });
    await vi.waitFor(() => {
      expect(deleteBotMock).toHaveBeenCalledTimes(1);
      expect(deleteButton.disabled).toBe(true);
    });
    resolveDelete?.({ ok: true });
    await vi.waitFor(() => {
      expect(navigateMock).toHaveBeenCalledWith("/dashboard/accounts");
      expect(toastMock).toHaveBeenCalledTimes(1);
    });
  });
});
