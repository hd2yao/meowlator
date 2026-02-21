import { finalizeInference, getUploadURL } from "../../utils/api";
import type { InferenceResult } from "../../types/shared";

function mockEdgeResult(): InferenceResult {
  return {
    intentTop3: [
      { label: "FEEDING", prob: 0.62 },
      { label: "SEEK_ATTENTION", prob: 0.21 },
      { label: "WANT_PLAY", prob: 0.09 },
    ],
    state: { tension: "MID", arousal: "MID", comfort: "LOW" },
    confidence: 0.62,
    source: "EDGE",
    evidence: ["靠近食盆区域", "尾巴摆动频率较高"],
    copyStyleVersion: "v1",
  };
}

Page({
  data: {
    loading: false,
    message: "",
  },

  async onSelectImage() {
    this.setData({ loading: true, message: "" });
    try {
      const app = getApp<{ globalData: { userId: string; catId: string; lastResult?: unknown } }>();
      const uploadMeta = await getUploadURL(app.globalData.userId, app.globalData.catId);
      await this.pickAndUpload(uploadMeta.uploadUrl);

      const response = await finalizeInference({
        sampleId: uploadMeta.sampleId,
        deviceCapable: true,
        sceneTag: "UNKNOWN",
        edgeResult: mockEdgeResult(),
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

  pickAndUpload(uploadUrl: string): Promise<void> {
    return new Promise((resolve, reject) => {
      wx.chooseMedia({
        count: 1,
        mediaType: ["image"],
        success: (mediaRes) => {
          const file = mediaRes.tempFiles[0];
          wx.uploadFile({
            url: uploadUrl,
            filePath: file.tempFilePath,
            name: "file",
            success: () => resolve(),
            fail: reject,
          });
        },
        fail: reject,
      });
    });
  },
});
