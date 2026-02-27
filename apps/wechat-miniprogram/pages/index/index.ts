import { finalizeInference, getAuthHeader, getClientConfig, getUploadURL } from "../../utils/api";
import { edgeInferenceEngine } from "../../utils/edge_inference";
import type { ClientConfig } from "../../utils/api";
import type { EdgeRuntime, InferenceResult } from "../../types/shared";

interface EdgeFinalizePayload {
  deviceCapable: boolean;
  edgeResult?: InferenceResult;
  edgeRuntime: EdgeRuntime;
}

interface AppGlobalData {
  catId: string;
  lastResult?: unknown;
  lastImagePath?: string;
}

Page({
  data: {
    loading: false,
    message: "",
  },

  async onTakePhoto() {
    await this.handleImageSelection(["camera"]);
  },

  async onPickFromAlbum() {
    await this.handleImageSelection(["album"]);
  },

  async handleImageSelection(sourceType: Array<"album" | "camera">) {
    this.setData({ loading: true, message: "" });
    try {
      const imagePath = await this.pickImage(sourceType);
      const clientConfig = await this.fetchClientConfig();
      const edgePayload = await this.runEdgeInference(imagePath, clientConfig);

      const app = getApp<{ globalData: AppGlobalData }>();
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
      app.globalData.lastImagePath = imagePath;
      wx.navigateTo({ url: "/pages/result/result" });
    } catch (err) {
      const message = err instanceof Error ? err.message : "识别失败，请稍后重试";
      this.setData({ message });
    } finally {
      this.setData({ loading: false });
    }
  },

  async fetchClientConfig(): Promise<ClientConfig | undefined> {
    try {
      return await getClientConfig();
    } catch (err) {
      return undefined;
    }
  },

  pickImage(sourceType: Array<"album" | "camera">): Promise<string> {
    return new Promise((resolve, reject) => {
      wx.chooseMedia({
        count: 1,
        mediaType: ["image"],
        sourceType,
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

  async runEdgeInference(imagePath: string, clientConfig?: ClientConfig): Promise<EdgeFinalizePayload> {
    if (clientConfig) {
      const selectedModel = clientConfig.modelRollout.selectedModel || clientConfig.modelVersion;
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
