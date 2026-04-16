# 术语字典与意图映射规范

> 日期：2026-02-25
> 版本：1.0.0
> 用途：统一所有文档、UI、代码中的术语与标签

---

## 1. 术语字典

### 推理相关

| 术语 | 标准表达 | 使用场景 | 示例 |
|------|---------|---------|------|
| 极速推理 | **极速推理** | UI、文案、文档 | "端侧极速推理 12ms" |
| 云端复判 / 云端推理 | **云端复判** | UI、文案、文档 | "云端复判 95%" |
| 端侧推理 | **端侧推理** | UI、文案、文档 | "端侧推理成功" |
| 启发式推理 | **启发式推理** | 技术文档 | "当前采用启发式推理" |
| ONNX 真推理 | **ONNX 推理** | 技术文档 | "切换到 ONNX 推理" |

### 意图相关

| 术语 | 标准表达 | 使用场景 |
|------|---------|---------|
| 意图 | **意图** | 所有场景 |
| 意图标签 | **意图标签** | 技术文档 |
| 意图分类 | **意图分类** | 技术文档 |
| 意图识别 | **意图识别** | 技术文档 |

### 推理结果相关

| 术语 | 标准表达 | 使用场景 |
|------|---------|---------|
| 置信度 | **置信度** | 所有场景 |
| 准确率 | **准确率** | 技术文档、指标 |
| 召回率 | **召回率** | 技术文档、指标 |
| 文案 | **文案** | UI、文档 |
| 拟人文案 | **拟人文案** | UI、文档 |

### 发布相关

| 术语 | 标准表达 | 使用场景 |
|------|---------|---------|
| 灰度发布 | **灰度发布** | 所有场景 |
| 灰度比例 | **灰度比例** | 技术文档 |
| 回滚 | **回滚** | 所有场景 |
| 模型版本 | **模型版本** | 所有场景 |

---

## 2. 意图映射表

### 标准意图枚举（后端）

```go
type IntentLabel string

const (
    IntentFeeding        IntentLabel = "FEEDING"
    IntentSeekAttention  IntentLabel = "SEEK_ATTENTION"
    IntentWantPlay       IntentLabel = "WANT_PLAY"
    IntentWantDoorOpen   IntentLabel = "WANT_DOOR_OPEN"
    IntentDefensiveAlert IntentLabel = "DEFENSIVE_ALERT"
    IntentRelaxSleep     IntentLabel = "RELAX_SLEEP"
    IntentCuriousObserve IntentLabel = "CURIOUS_OBSERVE"
    IntentUncertain      IntentLabel = "UNCERTAIN"
)
```

### 前端展示映射（TypeScript）

```typescript
interface IntentDisplay {
  emoji: string;           // 表情符号
  label: string;           // 反馈弹窗标签
  headline: string;        // 结果页展示文案
}

const INTENT_DISPLAY_MAP: Record<string, IntentDisplay> = {
  FEEDING: {
    emoji: '🍖',
    label: '要吃的',
    headline: '我想进食！'
  },
  SEEK_ATTENTION: {
    emoji: '👋',
    label: '求抚摸',
    headline: '我想贴贴！'
  },
  WANT_PLAY: {
    emoji: '⚽️',
    label: '想玩耍',
    headline: '我想玩耍！'
  },
  WANT_DOOR_OPEN: {
    emoji: '🚪',
    label: '要开门',
    headline: '快给我开门！'
  },
  DEFENSIVE_ALERT: {
    emoji: '😾',
    label: '警惕防御',
    headline: '我在防御警戒！'
  },
  RELAX_SLEEP: {
    emoji: '💤',
    label: '放松睡觉',
    headline: '我想安心睡会儿。'
  },
  CURIOUS_OBSERVE: {
    emoji: '👀',
    label: '好奇观察',
    headline: '我在观察情况。'
  },
  UNCERTAIN: {
    emoji: '❓',
    label: '摸鱼/不确定',
    headline: '我也说不准喵。'
  }
};
```

### 使用规范

**在反馈弹窗中**：使用 `label`
```
🍖 要吃的 (FEEDING)
👋 求抚摸 (SEEK_ATTENTION)
⚽️ 想玩耍 (WANT_PLAY)
🚪 要开门 (WANT_DOOR_OPEN)
😾 警惕防御 (DEFENSIVE_ALERT)
💤 放松睡觉 (RELAX_SLEEP)
👀 好奇观察 (CURIOUS_OBSERVE)
❓ 摸鱼/不确定 (UNCERTAIN)
```

**在结果页中**：使用 `headline`
```
我想进食！
我想贴贴！
我想玩耍！
快给我开门！
我在防御警戒！
我想安心睡会儿。
我在观察情况。
我也说不准喵。
```

**在 API 响应中**：使用 `emoji` + `headline`
```json
{
  "intentTop3": [
    {
      "label": "FEEDING",
      "prob": 0.62
    }
  ],
  "headline": "我想进食！"
}
```

---

## 3. 性能/成本基线（冻结版）

### 性能基准

| 指标 | 目标阈值 | 告警阈值 | 说明 |
|------|---------|---------|------|
| API 错误率 | < 1.5% | > 1.5%（持续 5 分钟） | API 错误率 |
| finalize p95 延迟 | < 500ms | > 2500ms（持续 5 分钟） | 推理响应时间 |
| 端侧推理 | < 100ms | - | 保守估计，包括模型加载 |
| 文案生成 | < 5s | - | 包括 LLM 调用 |
| 页面加载 | < 2s | - | 首屏渲染 |
| 首屏渲染 | < 1s | - | 小程序标准 |

### 成本预算（冻结版）

| 阶段 | 周成本 | 月成本 | 说明 |
|------|--------|--------|------|
| 第 1-2 周（高峰） | ¥8,000 | ¥16,000 | 含临时前端工程师 0.5 人 |
| 第 3 周后（稳定） | ¥3,000 | ¥12,000 | 正常运营成本 |
| 6 个月总预算 | - | ¥88,000 | 第 1-2 周 ¥16k + 第 3-26 周 ¥72k |

### 上线门禁（M6 ONNX 切换）

| 指标 | 门禁值 | 说明 |
|------|--------|------|
| 准确率 | ≥ 85% | 相对基线 |
| 错误率 | < 1.5% | API 错误率 |
| p95 延迟 | < 500ms | 推理响应时间 |
| 单次成本 | < ¥0.2 | 云端复判成本 |
| 云端兜底比例 | < 30% | 端侧命中率 > 70% |

---

## 4. 文档同步检查清单

### 每周同步检查

- [ ] docs/project_manual.md 与代码框架一致
- [ ] docs/project_manual.md 默认值与 config.go 一致
- [ ] docs/project_manual.md 表名与 SQL 迁移一致
- [ ] UI 文案与 INTENT_DISPLAY_MAP 一致
- [ ] 术语使用与术语字典一致
- [ ] 性能/成本基线与 KPI_BASELINE.md 一致

### 发布前检查

- [ ] docs/project_manual.md 章节顺序正确
- [ ] UI 文案与枚举映射完整
- [ ] 术语字典应用到所有文档
- [ ] 性能/成本基线冻结

---

## 5. 文件清单

| 文件 | 用途 | 维护频率 |
|------|------|---------|
| docs/project_manual.md | 项目手册（主文档） | 每周更新进度 |
| docs/api.md | API 文档 | 按需更新 |
| docs/TERMINOLOGY_AND_MAPPING.md | 术语字典 + 意图映射 | 按需更新 |
| docs/KPI_BASELINE.md | 性能/成本基线 | 冻结（不更新） |
| docs/plans/*.md | 阶段性执行计划 | 按需更新 |

---

## 6. 验收标准

- [ ] 所有术语使用一致
- [ ] 意图映射表完整且正确
- [ ] 性能/成本基线唯一且冻结
- [ ] 文档同步检查清单建立
- [ ] UI 设计稿与映射表一致
