const { finalizeInference, getAuthHeader, getClientConfig, getUploadURL } = require("../../utils/api");
const { edgeInferenceEngine } = require("../../utils/edge_inference");

Page({
  data: {
    loading: false,
    message: "",
  },

  async onSelectImage() {
    this.setData({ loading: true, message: "" });
    try {
      const imagePath = await this.pickImage();
      const clientConfig = await this.fetchClientConfig();
      const edgePayload = await this.runEdgeInference(imagePath, clientConfig);

      const app = getApp();
      const uploadMeta = await getUploadURL(app.globalData.catId);
      await this.uploadFile(uploadMeta.uploadUrl, imagePath);

      const response = await finalizeInference({
        sampleId: uploadMeta.sampleId,
        deviceCapable: edgePayload.deviceCapable,
        sceneTag: "UNKNOWN",
        edgeResult: edgePayload.edgeResult,
        edgeRuntime: edgePayload.edgeRuntime,
      });

      app.globalData.lastResult = response;
      wx.navigateTo({ url: "/pages/result/result" });
    } catch (err) {
      const message = err && err.message ? err.message : "识别失败，请稍后重试";
      this.setData({ message });
    } finally {
      this.setData({ loading: false });
    }
  },

  async fetchClientConfig() {
    try {
      return await getClientConfig();
    } catch (err) {
      return undefined;
    }
  },

  pickImage() {
    return new Promise((resolve, reject) => {
      wx.chooseMedia({
        count: 1,
        mediaType: ["image"],
        success: (mediaRes) => {
          const file = mediaRes.tempFiles[0];
          resolve(file.tempFilePath);
        },
        fail: reject,
      });
    });
  },

  uploadFile(uploadUrl, filePath) {
    return new Promise((resolve, reject) => {
      wx.uploadFile({
        url: uploadUrl,
        filePath,
        name: "file",
        header: getAuthHeader(),
        success: () => resolve(),
        fail: reject,
      });
    });
  },

  async runEdgeInference(imagePath, clientConfig) {
    if (clientConfig) {
      const selectedModel = (clientConfig.modelRollout && clientConfig.modelRollout.selectedModel) || clientConfig.modelVersion;
      edgeInferenceEngine.configure({
        modelVersion: selectedModel,
        modelHash: `cfg-${selectedModel}`,
      });
      if (!edgeInferenceEngine.isDeviceAllowed(clientConfig.edgeDeviceWhitelist || [])) {
        return {
          deviceCapable: false,
          edgeRuntime: edgeInferenceEngine.buildRuntime("device not in edge whitelist", 0, "DEVICE_NOT_WHITELISTED"),
        };
      }
    }

    let inferStartedAt = Date.now();
    try {
      await edgeInferenceEngine.loadModel();
      inferStartedAt = Date.now();
      const output = await edgeInferenceEngine.predict(imagePath);
      return {
        deviceCapable: true,
        edgeResult: output.result,
        edgeRuntime: output.runtime,
      };
    } catch (err) {
      const message = err && err.message ? err.message : "edge inference failed";
      const inferMs = Math.max(1, Date.now() - inferStartedAt);
      return {
        deviceCapable: false,
        edgeRuntime: edgeInferenceEngine.buildRuntime(message, inferMs),
      };
    }
  },

  onReady() {
    const health = edgeInferenceEngine.getHealth();
    if (!health.loaded) {
      edgeInferenceEngine.loadModel().catch(() => {
        return;
      });
    }
  },
});
