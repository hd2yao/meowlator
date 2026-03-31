import { beforeEach, describe, expect, it, vi } from "vitest";
import type { FinalizeResponse, InferenceResult } from "../types/shared";
import type { ClientConfig } from "../utils/api";

type PageDefinition = Record<string, any>;

const appState = {
  globalData: {
    catId: "cat-001",
    userId: "user-001",
    sessionToken: "session-001",
    sessionExpiresAt: Math.floor(Date.now() / 1000) + 3600,
    lastResult: undefined as FinalizeResponse | undefined,
    lastImagePath: undefined as string | undefined,
  },
};

const requestBehavior = {
  uploadUrlFail: false,
};

function makeClientConfig(overrides: Partial<ClientConfig> = {}): ClientConfig {
  return {
    edgeAcceptThreshold: 0.7,
    cloudFallbackThreshold: 0.45,
    copyStyleVersion: "v1",
    modelVersion: "mobilenetv3-small-int8-v1",
    abBucket: 0,
    shareTemplates: [],
    edgeDeviceWhitelist: [],
    modelRollout: {
      activeModel: "mobilenetv3-small-int8-v1",
      rolloutRatio: 0,
      targetBucket: 0,
      totalBuckets: 100,
    },
    riskEnabled: false,
    abBucketRules: { totalBuckets: 3 },
    ...overrides,
  };
}

function makeEdgeResult(): InferenceResult {
  return {
    intentTop3: [
      { label: "WANT_PLAY", prob: 0.93 },
      { label: "CURIOUS_OBSERVE", prob: 0.04 },
      { label: "UNCERTAIN", prob: 0.03 },
    ],
    state: { tension: "MID", arousal: "HIGH", comfort: "LOW" },
    confidence: 0.93,
    source: "EDGE",
    evidence: ["视觉主体明显"],
    copyStyleVersion: "v1",
  };
}

function makeFinalizeResponse(sampleId = "sample-123"): FinalizeResponse {
  return {
    sampleId,
    result: makeEdgeResult(),
    copy: { catLine: "喵", evidence: "证据", shareTitle: "share" },
    needFeedback: true,
    fallbackUsed: false,
  };
}

function installGlobals() {
  let capturedPage: PageDefinition | undefined;
  const wx = {
    chooseMedia: vi.fn(),
    uploadFile: vi.fn(),
    navigateTo: vi.fn(),
    showToast: vi.fn(),
    showShareMenu: vi.fn(),
    setClipboardData: vi.fn(),
    showModal: vi.fn(),
    navigateBack: vi.fn(),
    reLaunch: vi.fn(),
    login: vi.fn(),
    request: vi.fn((options: any) => {
      const url = String(options.url || "");
      if (url.includes("/v1/metrics/client-config")) {
        options.success?.({ statusCode: 200, data: makeClientConfig() });
        return;
      }
      if (url.includes("/v1/samples/upload-url")) {
        if (requestBehavior.uploadUrlFail) {
          options.fail?.(new Error("upload service down"));
          return;
        }
        options.success?.({
          statusCode: 200,
          data: {
            sampleId: "sample-123",
            imageKey: "samples/cat-001/sample-123.jpg",
            uploadUrl: "https://upload.example.local/put?key=samples/cat-001/sample-123.jpg",
            expiresInSeconds: 600,
            retentionDeadline: 1730000000,
          },
        });
        return;
      }
      if (url.includes("/v1/inference/finalize")) {
        options.success?.({
          statusCode: 200,
          data: makeFinalizeResponse(),
        });
        return;
      }
      options.success?.({ statusCode: 200, data: {} });
    }),
    getSystemInfoSync: vi.fn(() => ({ model: "iPhone15,2" })),
    getImageInfo: vi.fn((options: any) => {
      options.success?.({ width: 400, height: 300, path: options.src, orientation: "up", type: "jpg" });
    }),
  };

  vi.stubGlobal("wx", wx);
  vi.stubGlobal("getApp", () => appState);
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

async function loadIndexPage() {
  const env = installGlobals();
  await import("../pages/index/index");
  return { ...env, page: env.getPage() };
}

describe("index page", () => {
  beforeEach(() => {
    vi.resetModules();
    vi.clearAllMocks();
    appState.globalData.catId = "cat-001";
    appState.globalData.userId = "user-001";
    appState.globalData.sessionToken = "session-001";
    appState.globalData.sessionExpiresAt = Math.floor(Date.now() / 1000) + 3600;
    appState.globalData.lastResult = undefined;
    appState.globalData.lastImagePath = undefined;
    requestBehavior.uploadUrlFail = false;
  });

  it("runs the happy path from image selection to navigation", async () => {
    const { wx, page } = await loadIndexPage();

    wx.chooseMedia.mockImplementation((options: any) => {
      options.success({ tempFiles: [{ tempFilePath: "/tmp/cat.jpg" }] });
    });
    wx.uploadFile.mockImplementation((options: any) => {
      options.success({});
    });

    await page.handleImageSelection(["camera"]);

    expect(page.data.loading).toBe(false);
    expect(page.data.message).toBe("");
    expect(wx.request.mock.calls.map((call: any[]) => call[0].url)).toEqual([
      "http://127.0.0.1:8080/v1/metrics/client-config",
      "http://127.0.0.1:8080/v1/samples/upload-url",
      "http://127.0.0.1:8080/v1/inference/finalize",
    ]);
    expect(wx.request.mock.calls[1][0].data).toEqual({ catId: "cat-001", suffix: ".jpg" });
    expect(wx.request.mock.calls[2][0].data).toEqual(expect.objectContaining({
      sampleId: "sample-123",
      deviceCapable: true,
      sceneTag: "UNKNOWN",
    }));
    expect(appState.globalData.lastResult?.sampleId).toBe("sample-123");
    expect(appState.globalData.lastImagePath).toBe("/tmp/cat.jpg");
    expect(wx.navigateTo).toHaveBeenCalledWith({ url: "/pages/result/result" });
  });

  it("marks the device as not capable when the whitelist rejects it", async () => {
    const { page } = await loadIndexPage();

    const result = await page.runEdgeInference("/tmp/cat.jpg", makeClientConfig({ edgeDeviceWhitelist: ["Android"] }));

    expect(result.deviceCapable).toBe(false);
    expect(result.edgeResult).toBeUndefined();
    expect(result.edgeRuntime.failureCode).toBe("DEVICE_NOT_WHITELISTED");
  });

  it("shows an error message when upload-url fails", async () => {
    requestBehavior.uploadUrlFail = true;
    const { wx, page } = await loadIndexPage();

    wx.chooseMedia.mockImplementation((options: any) => {
      options.success({ tempFiles: [{ tempFilePath: "/tmp/cat.jpg" }] });
    });

    await page.handleImageSelection(["album"]);

    expect(page.data.loading).toBe(false);
    expect(page.data.message).toBe("upload service down");
    expect(wx.navigateTo).not.toHaveBeenCalled();
  });
});
