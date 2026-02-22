const BASE_URL = "http://127.0.0.1:8080";

function getGlobalData() {
  const app = getApp();
  return app.globalData;
}

function extractPath(url) {
  const match = String(url).match(/^https?:\/\/[^/]+(\/.*)$/);
  if (match && match[1]) {
    return match[1];
  }
  return String(url);
}

function fnv32Signature(method, path, ts, body, token) {
  let hash = 0x811c9dc5;
  const input = `${method}|${path}|${ts}|${body}|${token}`;
  for (let i = 0; i < input.length; i += 1) {
    hash ^= input.charCodeAt(i);
    hash = Math.imul(hash, 0x01000193);
  }
  return (hash >>> 0).toString(16).padStart(8, "0");
}

function rawRequest(options) {
  return new Promise((resolve, reject) => {
    wx.request({
      ...options,
      success: (res) => {
        if (res.statusCode >= 200 && res.statusCode < 300) {
          resolve(res.data);
          return;
        }
        const message = res.data && res.data.error ? String(res.data.error) : `request failed: ${res.statusCode}`;
        reject(new Error(message));
      },
      fail: reject,
    });
  });
}

function loginWeChat(code) {
  return rawRequest({
    url: `${BASE_URL}/v1/auth/wechat/login`,
    method: "POST",
    header: { "content-type": "application/json" },
    data: { code },
  });
}

function wxLoginCode() {
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

async function ensureSession() {
  const globalData = getGlobalData();
  const now = Math.floor(Date.now() / 1000);
  if (globalData.sessionToken && (globalData.sessionExpiresAt || 0) > now + 60) {
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

async function request(options, extra) {
  const requireAuth = !(extra && extra.requireAuth === false);
  const requireSignature = !!(extra && extra.requireSignature === true);
  if (requireAuth) {
    await ensureSession();
  }

  const globalData = getGlobalData();
  const headers = {
    ...((options && options.header) || {}),
  };

  if (requireAuth && globalData.sessionToken) {
    headers.Authorization = `Bearer ${globalData.sessionToken}`;
    headers["X-User-Id"] = globalData.userId;
  }

  if (requireSignature && globalData.sessionToken) {
    const method = ((options && options.method) || "GET").toUpperCase();
    const url = String((options && options.url) || "");
    const path = extractPath(url);
    const ts = String(Math.floor(Date.now() / 1000));
    const body = options && options.data != null
      ? (typeof options.data === "string" ? options.data : JSON.stringify(options.data))
      : "";
    headers["X-Req-Ts"] = ts;
    headers["X-Req-Sig"] = fnv32Signature(method, path, ts, body, globalData.sessionToken);
  }

  return rawRequest({
    ...options,
    header: headers,
  });
}

function getClientConfig() {
  return request({
    url: `${BASE_URL}/v1/metrics/client-config`,
    method: "GET",
  });
}

function getUploadURL(catId) {
  return request({
    url: `${BASE_URL}/v1/samples/upload-url`,
    method: "POST",
    header: { "content-type": "application/json" },
    data: { catId, suffix: ".jpg" },
  }, { requireSignature: true });
}

function finalizeInference(payload) {
  return request({
    url: `${BASE_URL}/v1/inference/finalize`,
    method: "POST",
    header: { "content-type": "application/json" },
    data: payload,
  });
}

function submitFeedback(payload) {
  return request({
    url: `${BASE_URL}/v1/feedback`,
    method: "POST",
    header: { "content-type": "application/json" },
    data: payload,
  });
}

function generateCopy(result) {
  return request({
    url: `${BASE_URL}/v1/copy/generate`,
    method: "POST",
    header: { "content-type": "application/json" },
    data: { result },
  });
}

function deleteSample(sampleId) {
  return request({
    url: `${BASE_URL}/v1/samples/${encodeURIComponent(sampleId)}`,
    method: "DELETE",
  }, { requireSignature: true });
}

function getAuthHeader() {
  const globalData = getGlobalData();
  if (!globalData.sessionToken) {
    return {};
  }
  return {
    Authorization: `Bearer ${globalData.sessionToken}`,
    "X-User-Id": globalData.userId,
  };
}

module.exports = {
  loginWeChat,
  ensureSession,
  getClientConfig,
  getUploadURL,
  finalizeInference,
  submitFeedback,
  generateCopy,
  deleteSample,
  getAuthHeader,
};
