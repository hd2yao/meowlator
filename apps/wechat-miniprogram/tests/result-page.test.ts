import { beforeEach, describe, expect, it, vi } from "vitest";
import type { FinalizeResponse } from "../types/shared";

type PageDefinition = Record<string, any>;

const appState = {
  globalData: {
    userId: "user-001",
    sessionToken: "session-001",
    sessionExpiresAt: Math.floor(Date.now() / 1000) + 3600,
    lastResult: undefined as FinalizeResponse | undefined,
    lastImagePath: undefined as string | undefined,
  },
};

function installGlobals() {
  let capturedPage: PageDefinition | undefined;
  const wx = {
    showShareMenu: vi.fn(),
    showToast: vi.fn(),
    showModal: vi.fn(),
    setClipboardData: vi.fn(),
    navigateBack: vi.fn(),
    reLaunch: vi.fn(),
    request: vi.fn((options: any) => {
      options.success?.({ statusCode: 200, data: {} });
    }),
  };

  vi.stubGlobal("wx", wx);
  vi.stubGlobal("getApp", () => appState);
  vi.stubGlobal("getCurrentPages", () => [{ route: "pages/index/index" }]);
  vi.stubGlobal("Page", (definition: PageDefinition) => {
    capturedPage = definition;
    return definition;
  });

  return {
    wx,
    getPage(): PageDefinition {
      if (!capturedPage) {
        throw new Error("page definition was not captured");
      }
      const page = {
        data: JSON.parse(JSON.stringify(capturedPage.data || {})),
        ...capturedPage,
        setData(patch: Record<string, unknown>) {
          Object.assign(page.data, patch);
        },
      };
      return page;
    },
  };
}

async function loadResultPage() {
  const env = installGlobals();
  await import("../pages/result/result");
  return { ...env, page: env.getPage() };
}

function makeFinalizeResponse(overrides: Partial<FinalizeResponse> = {}): FinalizeResponse {
  return {
    sampleId: "sample-123",
    result: {
      intentTop3: [
        { label: "WANT_PLAY", prob: 0.91 },
        { label: "CURIOUS_OBSERVE", prob: 0.05 },
        { label: "UNCERTAIN", prob: 0.04 },
      ],
      state: { tension: "MID", arousal: "HIGH", comfort: "LOW" },
      confidence: 0.91,
      source: "EDGE",
      evidence: ["视觉主体明显"],
      copyStyleVersion: "v1",
      edgeMeta: {
        engine: "wx-heuristic-v1",
        modelVersion: "mobilenetv3-small-int8-v2",
        loadMs: 11,
        inferMs: 19,
        deviceModel: "iPhone15,2",
        fallbackUsed: false,
        usedEdgeResult: true,
      },
      risk: {
        painRiskScore: 0.2,
        painRiskLevel: "LOW",
        riskEvidence: ["示例证据"],
        disclaimer: "非医疗诊断，仅作风险提示；若持续异常请咨询兽医。",
      },
    },
    copy: { catLine: "我想玩耍！", evidence: "视觉证据", shareTitle: "猫语翻译结果出炉" },
    needFeedback: true,
    fallbackUsed: false,
    ...overrides,
  };
}

describe("result page", () => {
  beforeEach(() => {
    vi.resetModules();
    vi.clearAllMocks();
    appState.globalData.userId = "user-001";
    appState.globalData.sessionToken = "session-001";
    appState.globalData.sessionExpiresAt = Math.floor(Date.now() / 1000) + 3600;
    appState.globalData.lastResult = undefined;
    appState.globalData.lastImagePath = undefined;
  });

  it("hydrates page state from the finalize response", async () => {
    const { page } = await loadResultPage();
    appState.globalData.lastResult = makeFinalizeResponse();
    appState.globalData.lastImagePath = "/tmp/cat.jpg";

    page.onLoad();

    expect(page.data.result?.source).toBe("EDGE");
    expect(page.data.imagePath).toBe("/tmp/cat.jpg");
    expect(page.data.intentCode).toBe("WANT_PLAY");
    expect(page.data.headline).toBe("我想玩耍！");
    expect(page.data.confidenceText).toBe("91%");
    expect(page.data.sourceLabel).toBe("端侧极速推理");
    expect(page.data.sourceLatency).toBe("19ms");
    expect(page.data.intentTop3Display).toHaveLength(3);
    expect(page.data.riskLevelText).toBe("LOW");
    expect(page.data.showFeedbackSheet).toBe(true);
  });

  it("opens the feedback sheet and accepts a valid label", async () => {
    const { page } = await loadResultPage();
    appState.globalData.lastResult = makeFinalizeResponse({ needFeedback: false });

    page.onLoad();
    page.onWrong();
    page.onPickGridLabel({ currentTarget: { dataset: { label: "FEEDING" } } });

    expect(page.data.showFeedbackSheet).toBe(true);
    expect(page.data.pickedLabel).toBe("FEEDING");
  });

  it("submits wrong feedback and closes the sheet", async () => {
    const { wx, page } = await loadResultPage();
    appState.globalData.lastResult = makeFinalizeResponse({ needFeedback: false });

    page.onLoad();
    page.setData({ sampleId: "sample-123", pickedLabel: "FEEDING", showFeedbackSheet: true });

    await page.onSubmitWrongFeedback();

    expect(page.data.showFeedbackSheet).toBe(false);
    expect(page.data.submittingFeedback).toBe(false);
    expect(wx.showToast).toHaveBeenCalledWith({ title: "反馈已记录", icon: "success" });
    expect(wx.request).toHaveBeenCalledWith(expect.objectContaining({
      url: "http://127.0.0.1:8080/v1/feedback",
      method: "POST",
      data: {
        sampleId: "sample-123",
        isCorrect: false,
        trueLabel: "FEEDING",
      },
    }));
  });

  it("blocks closing the feedback sheet while submitting", async () => {
    const { page } = await loadResultPage();
    appState.globalData.lastResult = makeFinalizeResponse({ needFeedback: false });

    page.onLoad();
    page.setData({ submittingFeedback: true, showFeedbackSheet: true });
    page.onCloseFeedbackSheet();

    expect(page.data.showFeedbackSheet).toBe(true);
  });
});
