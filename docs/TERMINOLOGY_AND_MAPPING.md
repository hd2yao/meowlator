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
  display: string;         // 结果页展示文案
  description: string;     // 详细描述
}

const INTENT_DISPLAY_MAP: Record<string, IntentDisplay> = {
  FEEDING: {
    emoji: '🍖',
    label: '要吃的',
    display: '进食',
    description: '猫咪想吃东西'
  },
  SEEK_ATTENTION: {
    emoji: '👋',
    label: '求抚摸',
    display: '寻求关注',
    description: '猫咪想要被抚摸或陪伴'
  },
  WANT_PLAY: {
    emoji: '⚽️',
    label: '想玩耍',
    display: '想玩耍',
    description: '猫咪想要玩耍'
  },
  WANT_DOOR_OPEN: {
    emoji: '🚪',
    label: '要开门',
    display: '想开门',
    description: '猫咪想要进出某个地方'
  },
  DEFENSIVE_ALERT: {
    emoji: '😾',
    label: '警惕防御',
    display: '防御警戒',
    description: '猫咪处于防御或警戒状态'
  },
  RELAX_SLEEP: {
    emoji: '💤',
    label: '放松睡觉',
    display: '放松睡眠',
    description: '猫咪想要放松或睡眠'
  },
  CURIOUS_OBSERVE: {
    emoji: '👀',
    label: '好奇观察',
    display: '好奇观察',
    description: '猫咪在好奇地观察周围'
  },
  UNCERTAIN: {
    emoji: '❓',
    label: '摸鱼/不确定',
    display: '不确定',
    description: '无法确定猫咪的意图'
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

**在结果页中**：使用 `display`
```
我想进食！
我想寻求关注！
我想玩耍！
我想开门！
我在防御警戒！
我想放松睡眠！
我在好奇观察！
我不确定...
```

**在 API 响应中**：使用 `emoji` + `display`
```json
{
  "intentTop3": [
    {
      "label": "FEEDING",
      "prob": 0.62
    }
  ],
  "display": "🍖 进食"
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

## 4. 资源模式选择

### 模式 A：完整团队（推荐）

**人员配置**：5.5 人
- 前端工程师：1.5 人（第 1 周）→ 1 人（第 2 周后）
- 后端工程师：1-2 人
- 机器学习工程师：1 人
- 运维工程师：0.5 人
- 产品经理：0.5 人
- QA 工程师：0.5 人

**治理流程**：
- 每日站会（15 分钟）
- 周报与评审
- 问题跟踪与升级（P0/P1/P2/P3）
- 变更管理流程
- 多角色审批

**适用场景**：正式项目、多人团队

---

### 模式 B：单人冲刺

**人员配置**：1-2 人
- 全栈工程师：1 人（主要）
- 支持工程师：1 人（兼职）

**治理流程**（简化）：
- 每日自检清单（替代站会）
- 周末总结（替代周报）
- 自检清单（替代多角色审批）
- 问题跟踪（简化为 P0/P1）

**自检清单示例**：
```
【每日自检】
- [ ] 代码编写进度
- [ ] 单元测试通过
- [ ] 代码审查（自审）
- [ ] 阻塞问题记录
- [ ] 风险预警

【周末总结】
- [ ] 本周完成的里程碑
- [ ] 下周计划
- [ ] 风险与问题
- [ ] 成本消耗
```

**适用场景**：MVP 冲刺、单人或小团队

---

## 5. 文档同步检查清单

### 每周同步检查

- [ ] research.md 与代码框架一致（net/http vs Gin）
- [ ] research.md 默认值与 config.go 一致
- [ ] research.md 表名与 SQL 迁移一致
- [ ] UI 文案与 INTENT_DISPLAY_MAP 一致
- [ ] 术语使用与术语字典一致
- [ ] 性能/成本基线与 KPI_BASELINE.md 一致

### 发布前检查

- [ ] plan.md 章节顺序正确
- [ ] research.md 事实校正完成
- [ ] UI 文案与枚举映射完整
- [ ] 术语字典应用到所有文档
- [ ] 资源模式选择明确
- [ ] 性能/成本基线冻结

---

## 6. 文件清单

| 文件 | 用途 | 维护频率 |
|------|------|---------|
| plan.md | 执行主计划 | 每周更新进度 |
| research.md | 技术参考 | 每周同步代码 |
| TERMINOLOGY_AND_MAPPING.md | 术语字典 + 意图映射 | 按需更新 |
| KPI_BASELINE.md | 性能/成本基线 | 冻结（不更新） |
| RESOURCE_MODES.md | 资源模式 | 按需更新 |
| （待抽离）前端映射代码文件 | 当前尚未独立成文件，先以本文档示例为准 | 与本文档同步 |

---

## 7. 验收标准

- [ ] 所有术语使用一致
- [ ] 意图映射表完整且正确
- [ ] 性能/成本基线唯一且冻结
- [ ] 资源模式 A/B 清晰可选
- [ ] 文档同步检查清单建立
- [ ] 前端映射文件抽离计划已记录（当前示例以本文档为准）
- [ ] UI 设计稿与映射表一致
