import type { EdgeRuntime, FinalizeResponse, InferenceResult, IntentLabel } from "../types/shared";

const BASE_URL = "http://127.0.0.1:8080";

interface UploadURLResponse {
  sampleId: string;
  imageKey: string;
  uploadUrl: string;
  expiresInSeconds: number;
  retentionDeadline: number;
}

interface WechatLoginResponse {
  userId: string;
  sessionToken: string;
  expiresAt: number;
}

interface ClientConfig {
  edgeAcceptThreshold: number;
  cloudFallbackThreshold: number;
  copyStyleVersion: string;
  modelVersion: string;
  abBucket: number;
  shareTemplates: string[];
  edgeDeviceWhitelist: string[];
  modelRollout: {
    activeModel: string;
    rolloutRatio: number;
    targetBucket: number;
  };
  riskEnabled: boolean;
  abBucketRules: { totalBuckets: number };
}

interface GlobalData {
  userId: string;
  catId: string;
  sessionToken?: string;
  sessionExpiresAt?: number;
  authPromise?: Promise<void>;
}

function getGlobalData(): GlobalData {
  const app = getApp<{ globalData: GlobalData }>();
  return app.globalData;
}

function extractPath(url: string): string {
  const match = url.match(/^https?:\/\/[^/]+(\/.*)$/);
  if (match && match[1]) {
    return match[1];
  }
  return url;
}

function fnv32Signature(method: string, path: string, ts: string, body: string, token: string): string {
  let hash = 0x811c9dc5;
  const input = `${method}|${path}|${ts}|${body}|${token}`;
  for (let i = 0; i < input.length; i += 1) {
    hash ^= input.charCodeAt(i);
    hash = Math.imul(hash, 0x01000193);
  }
  return (hash >>> 0).toString(16).padStart(8, "0");
}

function rawRequest<T>(options: WechatMiniprogram.RequestOption): Promise<T> {
  return new Promise((resolve, reject) => {
    wx.request({
      ...options,
      success: (res) => {
        if (res.statusCode >= 200 && res.statusCode < 300) {
          resolve(res.data as T);
          return;
        }
        const message = typeof res.data === "object" && res.data && "error" in (res.data as Record<string, unknown>)
          ? String((res.data as Record<string, unknown>).error)
          : `request failed: ${res.statusCode}`;
        reject(new Error(message));
      },
      fail: reject,
    });
  });
}

export async function loginWeChat(code: string): Promise<WechatLoginResponse> {
  return rawRequest<WechatLoginResponse>({
    url: `${BASE_URL}/v1/auth/wechat/login`,
    method: "POST",
    header: { "content-type": "application/json" },
    data: { code },
  });
}

function wxLoginCode(): Promise<string> {
  return new Promise((resolve) => {
    wx.login({
      success: (res) => {
        if (res.code) {
          resolve(res.code);
          return;
        }
        resolve(`dev-code-${Date.now()}`);
      },
      fail: () => resolve(`dev-code-${Date.now()}`),
    });
  });
}

export async function ensureSession(): Promise<void> {
  const globalData = getGlobalData();
  const now = Math.floor(Date.now() / 1000);
  if (globalData.sessionToken && (globalData.sessionExpiresAt || 0) > now+60) {
    return;
  }
  if (globalData.authPromise) {
    await globalData.authPromise;
    return;
  }

  globalData.authPromise = (async () => {
    const code = await wxLoginCode();
    const resp = await loginWeChat(code);
    globalData.userId = resp.userId;
    globalData.sessionToken = resp.sessionToken;
    globalData.sessionExpiresAt = resp.expiresAt;
  })();

  try {
    await globalData.authPromise;
  } finally {
    globalData.authPromise = undefined;
  }
}

async function request<T>(
  options: WechatMiniprogram.RequestOption,
  extra?: { requireAuth?: boolean; requireSignature?: boolean }
): Promise<T> {
  const requireAuth = extra?.requireAuth !== false;
  const requireSignature = extra?.requireSignature === true;

  if (requireAuth) {
    await ensureSession();
  }

  const globalData = getGlobalData();
  const headers: Record<string, string> = {
    ...(options.header as Record<string, string> || {}),
  };

  if (requireAuth && globalData.sessionToken) {
    headers["Authorization"] = `Bearer ${globalData.sessionToken}`;
    headers["X-User-Id"] = globalData.userId;
  }

  if (requireSignature && globalData.sessionToken) {
    const method = (options.method || "GET").toUpperCase();
    const url = String(options.url || "");
    const path = extractPath(url);
    const ts = String(Math.floor(Date.now() / 1000));
    const body = options.data == null
      ? ""
      : (typeof options.data === "string" ? options.data : JSON.stringify(options.data));
    headers["X-Req-Ts"] = ts;
    headers["X-Req-Sig"] = fnv32Signature(method, path, ts, body, globalData.sessionToken);
  }

  return rawRequest<T>({
    ...options,
    header: headers,
  });
}

export function getClientConfig(): Promise<ClientConfig> {
  return request<ClientConfig>({
    url: `${BASE_URL}/v1/metrics/client-config`,
    method: "GET",
  });
}

export function getUploadURL(catId: string): Promise<UploadURLResponse> {
  return request<UploadURLResponse>({
    url: `${BASE_URL}/v1/samples/upload-url`,
    method: "POST",
    header: { "content-type": "application/json" },
    data: { catId, suffix: ".jpg" },
  }, { requireSignature: true });
}

export function finalizeInference(payload: {
  sampleId: string;
  deviceCapable: boolean;
  sceneTag: string;
  edgeResult?: InferenceResult;
  edgeRuntime?: EdgeRuntime;
}): Promise<FinalizeResponse> {
  return request<FinalizeResponse>({
    url: `${BASE_URL}/v1/inference/finalize`,
    method: "POST",
    header: { "content-type": "application/json" },
    data: payload,
  });
}

export function submitFeedback(payload: {
  sampleId: string;
  isCorrect: boolean;
  trueLabel?: IntentLabel;
}): Promise<unknown> {
  return request<unknown>({
    url: `${BASE_URL}/v1/feedback`,
    method: "POST",
    header: { "content-type": "application/json" },
    data: payload,
  });
}

export function generateCopy(result: InferenceResult): Promise<{ catLine: string; evidence: string; shareTitle: string }> {
  return request<{ catLine: string; evidence: string; shareTitle: string }>({
    url: `${BASE_URL}/v1/copy/generate`,
    method: "POST",
    header: { "content-type": "application/json" },
    data: { result },
  });
}

export function deleteSample(sampleId: string): Promise<{ deleted: string }> {
  return request<{ deleted: string }>({
    url: `${BASE_URL}/v1/samples/${encodeURIComponent(sampleId)}`,
    method: "DELETE",
  }, { requireSignature: true });
}

export function getAuthHeader(): Record<string, string> {
  const globalData = getGlobalData();
  if (!globalData.sessionToken) {
    return {};
  }
  return {
    Authorization: `Bearer ${globalData.sessionToken}`,
    "X-User-Id": globalData.userId,
  };
}
