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

interface IntentDisplay {
  emoji: string;
  label: string;
  headline: string;
}

const INTENT_DISPLAY_MAP: Record<IntentLabel, IntentDisplay> = {
  FEEDING: { emoji: "🍖", label: "要吃的", headline: "我想进食！" },
  SEEK_ATTENTION: { emoji: "👋", label: "求抚摸", headline: "我想贴贴！" },
  WANT_PLAY: { emoji: "⚽️", label: "想玩耍", headline: "我想玩耍！" },
  WANT_DOOR_OPEN: { emoji: "🚪", label: "要开门", headline: "快给我开门！" },
  DEFENSIVE_ALERT: { emoji: "😾", label: "警惕防御", headline: "我在防御警戒！" },
  RELAX_SLEEP: { emoji: "💤", label: "放松睡觉", headline: "我想安心睡会儿。" },
  CURIOUS_OBSERVE: { emoji: "👀", label: "好奇观察", headline: "我在观察情况。" },
  UNCERTAIN: { emoji: "❓", label: "摸鱼/不确定", headline: "我也说不准喵。" },
};

const FEEDBACK_OPTIONS = INTENTS.map((value) => ({
  value,
  emoji: INTENT_DISPLAY_MAP[value].emoji,
  label: INTENT_DISPLAY_MAP[value].label,
}));

interface AppGlobalData {
  lastResult?: FinalizeResponse;
  lastImagePath?: string;
}

Page({
  data: {
    result: null as FinalizeResponse["result"] | null,
    imagePath: "",
    intentCode: "",
    headline: "",
    confidenceText: "0%",
    sourceLabel: "端侧极速推理",
    sourceLatency: "12ms",
    copy: { catLine: "", evidence: "", shareTitle: "" },
    intentTop3Display: [] as Array<{ label: string; probPercent: number }>,
    riskLevelText: "",
    riskScoreText: "",
    riskEvidenceText: "",
    riskDisclaimerText: "",
    sampleId: "",
    showFeedbackSheet: false,
    feedbackOptions: FEEDBACK_OPTIONS,
    pickedLabel: "",
    submittingFeedback: false,
  },

  onLoad() {
    wx.showShareMenu({
      withShareTicket: true,
      fail: () => {
        return;
      },
    });

    const app = getApp<{ globalData: AppGlobalData }>();
    const payload = app.globalData.lastResult;
    if (!payload) {
      wx.showToast({ title: "暂无结果", icon: "none" });
      return;
    }

    const top1Label = payload.result.intentTop3?.[0]?.label || "UNCERTAIN";
    const top1 = INTENT_DISPLAY_MAP[top1Label];
    const sourceLabel = payload.result.source === "CLOUD" ? "云端复判" : "端侧极速推理";
    const inferMs = payload.result.edgeMeta?.inferMs || 0;
    const sourceLatency = `${inferMs > 0 ? inferMs : payload.result.source === "CLOUD" ? 286 : 12}ms`;

    const intentTop3Display = (payload.result.intentTop3 || []).map((item) => {
      const display = INTENT_DISPLAY_MAP[item.label] || INTENT_DISPLAY_MAP.UNCERTAIN;
      return {
        label: `${display.emoji} ${display.label}`,
        probPercent: Math.round((item.prob || 0) * 100),
      };
    });

    const risk = payload.result.risk;
    this.setData({
      result: payload.result,
      imagePath: app.globalData.lastImagePath || "",
      intentCode: top1Label,
      headline: top1.headline,
      confidenceText: `${Math.round((payload.result.confidence || 0) * 100)}%`,
      sourceLabel,
      sourceLatency,
      copy: payload.copy,
      intentTop3Display,
      riskLevelText: risk ? risk.painRiskLevel : "",
      riskScoreText: risk ? `${Math.round((risk.painRiskScore || 0) * 100)}%` : "",
      riskEvidenceText: risk ? (risk.riskEvidence || []).join("、") : "",
      riskDisclaimerText: risk ? risk.disclaimer : "",
      sampleId: payload.sampleId,
      showFeedbackSheet: payload.needFeedback,
    });
  },

  onBackTap() {
    const pages = getCurrentPages();
    if (pages.length > 1) {
      wx.navigateBack({ delta: 1 });
      return;
    }
    wx.reLaunch({ url: "/pages/index/index" });
  },

  noop() {
    return;
  },

  onWrong() {
    this.setData({ showFeedbackSheet: true });
  },

  onCloseFeedbackSheet() {
    if (this.data.submittingFeedback) {
      return;
    }
    this.setData({ showFeedbackSheet: false });
  },

  onPickGridLabel(e: WechatMiniprogram.BaseEvent) {
    const currentTarget = e.currentTarget as WechatMiniprogram.BaseEvent["currentTarget"] & {
      dataset?: { label?: string };
    };
    const rawLabel = currentTarget.dataset?.label || "";
    if (!INTENTS.includes(rawLabel as IntentLabel)) {
      return;
    }
    this.setData({ pickedLabel: rawLabel });
  },

  async onCorrect() {
    await this.sendFeedback(true);
  },

  async onSubmitWrongFeedback() {
    if (!this.data.pickedLabel) {
      wx.showToast({ title: "先选真实意图", icon: "none" });
      return;
    }
    await this.sendFeedback(false, this.data.pickedLabel as IntentLabel);
  },

  async sendFeedback(isCorrect: boolean, trueLabel?: IntentLabel) {
    if (!this.data.sampleId || this.data.submittingFeedback) {
      return;
    }
    this.setData({ submittingFeedback: true });
    try {
      await submitFeedback({
        sampleId: this.data.sampleId,
        isCorrect,
        trueLabel,
      });
      wx.showToast({ title: "反馈已记录", icon: "success" });
      this.setData({ showFeedbackSheet: false });
    } catch (err) {
      wx.showToast({ title: "反馈失败", icon: "none" });
    } finally {
      this.setData({ submittingFeedback: false });
    }
  },

  onShareError() {
    wx.showModal({
      title: "暂不可用",
      content: "当前小程序未完成认证，系统分享不可用。可先复制分享文案并截图转发。",
      confirmText: "复制文案",
      cancelText: "知道了",
      success: (res) => {
        if (!res.confirm) {
          return;
        }
        this.onCopyShareTitle();
      },
    });
  },

  onCopyShareTitle() {
    const title = this.data.copy.shareTitle || "猫语翻译结果出炉";
    wx.setClipboardData({
      data: title,
      success: () => {
        wx.showToast({ title: "分享文案已复制", icon: "none" });
      },
      fail: () => {
        wx.showToast({ title: "复制失败，请重试", icon: "none" });
      },
    });
  },

  onShareAppMessage() {
    return {
      title: this.data.copy.shareTitle || "猫语翻译结果出炉",
      path: "/pages/index/index",
    };
  },
});
