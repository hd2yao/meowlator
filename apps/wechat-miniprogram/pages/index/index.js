"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
const api_1 = require("../../utils/api");
const edge_inference_1 = require("../../utils/edge_inference");
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
    async handleImageSelection(sourceType) {
        this.setData({ loading: true, message: "" });
        try {
            const imagePath = await this.pickImage(sourceType);
            const clientConfig = await this.fetchClientConfig();
            const edgePayload = await this.runEdgeInference(imagePath, clientConfig);
            const app = getApp();
            const uploadMeta = await (0, api_1.getUploadURL)(app.globalData.catId);
            await this.uploadFile(uploadMeta.uploadUrl, imagePath);
            const response = await (0, api_1.finalizeInference)({
                sampleId: uploadMeta.sampleId,
                deviceCapable: edgePayload.deviceCapable,
                sceneTag: "UNKNOWN",
                edgeResult: edgePayload.edgeResult,
                edgeRuntime: edgePayload.edgeRuntime,
            });
            app.globalData.lastResult = response;
            app.globalData.lastImagePath = imagePath;
            wx.navigateTo({ url: "/pages/result/result" });
        }
        catch (err) {
            const message = err instanceof Error ? err.message : "识别失败，请稍后重试";
            this.setData({ message });
        }
        finally {
            this.setData({ loading: false });
        }
    },
    async fetchClientConfig() {
        try {
            return await (0, api_1.getClientConfig)();
        }
        catch (err) {
            return undefined;
        }
    },
    pickImage(sourceType) {
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
    uploadFile(uploadUrl, filePath) {
        return new Promise((resolve, reject) => {
            wx.uploadFile({
                url: uploadUrl,
                filePath,
                name: "file",
                header: (0, api_1.getAuthHeader)(),
                success: () => resolve(),
                fail: reject,
            });
        });
    },
    async runEdgeInference(imagePath, clientConfig) {
        if (clientConfig) {
            const selectedModel = clientConfig.modelRollout.selectedModel || clientConfig.modelVersion;
            edge_inference_1.edgeInferenceEngine.configure({
                modelVersion: selectedModel,
                modelHash: `cfg-${selectedModel}`,
            });
            if (!edge_inference_1.edgeInferenceEngine.isDeviceAllowed(clientConfig.edgeDeviceWhitelist || [])) {
                return {
                    deviceCapable: false,
                    edgeRuntime: edge_inference_1.edgeInferenceEngine.buildRuntime("device not in edge whitelist", 0, "DEVICE_NOT_WHITELISTED"),
                };
            }
        }
        let inferStartedAt = Date.now();
        try {
            await edge_inference_1.edgeInferenceEngine.loadModel();
            inferStartedAt = Date.now();
            const output = await edge_inference_1.edgeInferenceEngine.predict(imagePath);
            return {
                deviceCapable: true,
                edgeResult: output.result,
                edgeRuntime: output.runtime,
            };
        }
        catch (err) {
            const message = err instanceof Error ? err.message : "edge inference failed";
            const inferMs = Math.max(1, Date.now() - inferStartedAt);
            return {
                deviceCapable: false,
                edgeRuntime: edge_inference_1.edgeInferenceEngine.buildRuntime(message, inferMs),
            };
        }
    },
    onReady() {
        const health = edge_inference_1.edgeInferenceEngine.getHealth();
        if (!health.loaded) {
            edge_inference_1.edgeInferenceEngine.loadModel().catch(() => {
                return;
            });
        }
    },
});
