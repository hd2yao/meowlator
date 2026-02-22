import type { EdgeRuntime, FinalizeResponse, InferenceResult, IntentLabel } from "../types/shared";

const BASE_URL = "http://127.0.0.1:8080";

interface UploadURLResponse {
  sampleId: string;
  imageKey: string;
  uploadUrl: string;
  expiresInSeconds: number;
  retentionDeadline: number;
}

interface ClientConfig {
  edgeAcceptThreshold: number;
  cloudFallbackThreshold: number;
  copyStyleVersion: string;
  modelVersion: string;
  abBucket: number;
  shareTemplates: string[];
}

function request<T>(options: WechatMiniprogram.RequestOption): Promise<T> {
  return new Promise((resolve, reject) => {
    wx.request({
      ...options,
      success: (res) => {
        if (res.statusCode >= 200 && res.statusCode < 300) {
          resolve(res.data as T);
          return;
        }
        reject(new Error(`request failed: ${res.statusCode}`));
      },
      fail: reject,
    });
  });
}

export function getClientConfig(userId: string): Promise<ClientConfig> {
  return request<ClientConfig>({
    url: `${BASE_URL}/v1/metrics/client-config?userId=${encodeURIComponent(userId)}`,
    method: "GET",
  });
}

export function getUploadURL(userId: string, catId: string): Promise<UploadURLResponse> {
  return request<UploadURLResponse>({
    url: `${BASE_URL}/v1/samples/upload-url`,
    method: "POST",
    header: { "content-type": "application/json" },
    data: { userId, catId, suffix: ".jpg" },
  });
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
  userId: string;
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
