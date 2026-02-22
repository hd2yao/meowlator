import { submitFeedback } from "../../utils/api";
import type { FinalizeResponse, IntentLabel } from "../../types/shared";

const INTENTS: IntentLabel[] = [
  "FEEDING",
  "SEEK_ATTENTION",
  "WANT_PLAY",
  "WANT_DOOR_OPEN",
  "DEFENSIVE_ALERT",
  "RELAX_SLEEP",
  "CURIOUS_OBSERVE",
  "UNCERTAIN",
];

Page({
  data: {
    result: null as FinalizeResponse["result"] | null,
    intentTop3Display: [] as Array<{ label: string; probPercent: number }>,
    copy: { catLine: "", evidence: "", shareTitle: "" },
    riskLevelText: "",
    riskScoreText: "",
    riskEvidenceText: "",
    riskDisclaimerText: "",
    sampleId: "",
    showLabelPicker: false,
    intentOptions: INTENTS,
    pickedLabel: "",
    pickedLabelText: "点击选择",
  },

  onLoad() {
    const app = getApp<{ globalData: { lastResult?: FinalizeResponse } }>();
    const payload = app.globalData.lastResult;
    if (!payload) {
      wx.showToast({ title: "暂无结果", icon: "none" });
      return;
    }
    const intentTop3Display = (payload.result.intentTop3 || []).map((item) => ({
      label: item.label,
      probPercent: Math.round((item.prob || 0) * 100),
    }));
    const risk = payload.result.risk;
    this.setData({
      result: payload.result,
      intentTop3Display,
      copy: payload.copy,
      riskLevelText: risk ? risk.painRiskLevel : "",
      riskScoreText: risk ? `${Math.round(risk.painRiskScore * 100)}%` : "",
      riskEvidenceText: risk ? (risk.riskEvidence || []).join("、") : "",
      riskDisclaimerText: risk ? risk.disclaimer : "",
      sampleId: payload.sampleId,
      showLabelPicker: payload.needFeedback,
    });
  },

  async onCorrect() {
    await this.sendFeedback(true);
  },

  async onWrong() {
    if (!this.data.pickedLabel) {
      this.setData({ showLabelPicker: true });
      wx.showToast({ title: "先选真实意图", icon: "none" });
      return;
    }
    await this.sendFeedback(false, this.data.pickedLabel as IntentLabel);
  },

  onPickTrueLabel(e: WechatMiniprogram.PickerChange) {
    const index = Number(e.detail.value);
    this.setData({ pickedLabel: INTENTS[index], pickedLabelText: INTENTS[index] });
  },

  async sendFeedback(isCorrect: boolean, trueLabel?: IntentLabel) {
    try {
      await submitFeedback({
        sampleId: this.data.sampleId,
        isCorrect,
        trueLabel,
      });
      wx.showToast({ title: "反馈已记录", icon: "success" });
    } catch (err) {
      wx.showToast({ title: "反馈失败", icon: "none" });
    }
  },

  onShareTap() {
    wx.showShareMenu({ withShareTicket: true });
    wx.showToast({ title: "可直接右上角分享", icon: "none" });
  },

  onShareAppMessage() {
    return {
      title: this.data.copy.shareTitle || "猫语翻译结果出炉",
      path: "/pages/index/index",
    };
  },
});
