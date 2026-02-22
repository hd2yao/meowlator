const BASE_URL = "http://127.0.0.1:8080";

function request(options) {
  return new Promise((resolve, reject) => {
    wx.request({
      ...options,
      success: (res) => {
        if (res.statusCode >= 200 && res.statusCode < 300) {
          resolve(res.data);
          return;
        }
        reject(new Error(`request failed: ${res.statusCode}`));
      },
      fail: reject,
    });
  });
}

function getClientConfig(userId) {
  return request({
    url: `${BASE_URL}/v1/metrics/client-config?userId=${encodeURIComponent(userId)}`,
    method: "GET",
  });
}

function getUploadURL(userId, catId) {
  return request({
    url: `${BASE_URL}/v1/samples/upload-url`,
    method: "POST",
    header: { "content-type": "application/json" },
    data: { userId, catId, suffix: ".jpg" },
  });
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

module.exports = {
  getClientConfig,
  getUploadURL,
  finalizeInference,
  submitFeedback,
  generateCopy,
};
