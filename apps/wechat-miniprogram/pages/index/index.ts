import { finalizeInference, getAuthHeader, getUploadURL } from "../../utils/api";
import { edgeInferenceEngine } from "../../utils/edge_inference";
import type { EdgeRuntime, InferenceResult } from "../../types/shared";

interface EdgeFinalizePayload {
  deviceCapable: boolean;
  edgeResult?: InferenceResult;
  edgeRuntime: EdgeRuntime;
}

Page({
  data: {
    loading: false,
    message: "",
  },

  async onSelectImage() {
    this.setData({ loading: true, message: "" });
    try {
      const imagePath = await this.pickImage();
      const edgePayload = await this.runEdgeInference(imagePath);

      const app = getApp<{ globalData: { catId: string; lastResult?: unknown } }>();
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
      const message = err instanceof Error ? err.message : "识别失败，请稍后重试";
      this.setData({ message });
    } finally {
      this.setData({ loading: false });
    }
  },

  pickImage(): Promise<string> {
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

  uploadFile(uploadUrl: string, filePath: string): Promise<void> {
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

  async runEdgeInference(imagePath: string): Promise<EdgeFinalizePayload> {
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
      const message = err instanceof Error ? err.message : "edge inference failed";
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
