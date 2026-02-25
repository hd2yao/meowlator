# Meowlator Mini Program UI/UX Design Specification

> **说明**：本文档包含 Meowlator 小程序端侧的高保真 UI 设计规范、设计令牌（Design Tokens）以及视觉布局要求。为了方便直接导入 Figma，我们同时提供了一份 Tailwind HTML 源码文件：`docs/ui_design.html`。

---

## 🎨 1. 设计令牌 (Design Tokens)

### 1.1 品牌色彩 (Brand Colors)
我们选用 **活力橙 (Vibrant Cat Orange)** 作为主品牌色，温暖且具有互动感。

- `Primary`: **#FF8A4C** (核心行为按钮、高亮意图文本)
- `Primary Dark`: **#E06B2E** (按钮按压态)
- `Secondary`: **#FCD34D** (辅助暖黄色，用于插画或强调)
- `Surface`: **#FFFFFF** (卡片白)
- `Background`: **#FAFAFA** (小程序底层背景灰/白)
- `Text Primary`: **#1F2937** (主标题、强调文案)
- `Text Secondary`: **#6B7280** (次要提示信息)
- `Success`: **#10B981** (准确反馈按钮)
- `Danger/Warning`: **#EF4444** (报错、不准反馈按钮)

### 1.2 字体排版 (Typography)
推荐采用无级变化的系统级黑体，以确保在 iOS 和 Android 端侧渲染的清晰度。英文推荐 `Inter`，中文使用系统默认屏显黑体。

- **大标题 (Hero Title)**: `28pt/36px`, Extrabold, 文字颜色: Text Primary
- **卡片标题 (Card Title)**: `18pt/24px`, Bold, 文字颜色: Text Primary
- **正文内容 (Body)**: `15pt/22px`, Regular, 文字颜色: Text Primary/Secondary
- **微小提示 (Caption)**: `12pt/16px`, Regular, 文字颜色: Text Secondary

### 1.3 圆角与投影 (Radii & Shadows)
引入现代大圆角卡片流设计，降低生硬感。

- **卡片/按钮圆角**: `16px`、`24px` 或 `32px`
- **卡片投影 (Soft Shadow)**: `0 10px 40px -10px rgba(0,0,0,0.08)` (非常柔和，提升层级感但不过分厚重)

---

## 📱 2. 核心页面说明 (Screens)

### 2.1 首页：拍照与上传 (Home Screen)
负责唤起用户的初次使用欲望。

- **头部导航**: 隐藏原生 Navigation Bar 标题，采用自定义空 Header，以释放更多沉浸空间。
- **大字号 Slogan**: "读懂猫咪的每一个动作" (采用品牌色高亮关键词)。
- **操作区 (Drop Zone)**: 占据屏幕主视觉的大卡片（1:1 或 4:5 比例），中间放置猫咪 Emoji 或品牌 IP 插画。
- **两个核心按钮**:
  1. `📸 立即拍照` (Primary Color 填充 + 强投影，视觉最重)
  2. `🖼️ 从相册选择` (浅灰背景，视觉次之)
- **合规提示**: 底部需置灰小字（12px），展现“非医疗诊断，照片7天销毁”。

### 2.2 结果页 (Result Screen)
展示端侧推理/云端兜底的结果。布局上将结果卡片“覆盖”在图片上半透明浮层。

- **上半部分 (Image Header)**: 用户上传/拍摄的猫咪照片满宽展示（高度约占据 45%），左上角带后退按钮。照片左下角浮动显示“端侧极速推理 12ms”等透明状态标签。
- **下半部分 (Result Card)**: 从底部升起的大圆角（32px）白色卡片。
  - **来源标签互斥规则**: 同一条结果只显示一种来源标签，`端侧极速推理` 或 `云端复判` 二选一，不可同屏并存。
  - **意图标签 (Intent Tag)**: 如 "WANT_PLAY"，全大写，主色点睛。
  - **翻译文本标题**: 放大且极具冲击力的中文翻译，如 "我想玩耍！"
  - **置信度提示**: 卡片右上角用小块强调展示推理置信度（如 95%）。
  - **AI 拟人文案块**: 内嵌浅灰底色卡片存放拟人搞笑文本，增加趣味性。
  - **验证反馈区**: 底部并排两个大按钮 (✅ 超准 和 ❌ 瞎说)，用于后续主动学习的数据回流。

### 2.3 纠错反馈弹窗 (Feedback Action Sheet)
当用户在结果页点击“❌ 瞎说”时，从底部弹出的半屏结构。

- **背板处理**: 原页面 40% 黑色半透明遮罩叠加。
- **Title**: “纠正意图” (带关闭 ✕ 按钮)
- **选择网格**: 2列 × 4行 的标签网格，枚举所有 8 类意图。当选中时采用 Primary 细边框或浅色底色高亮。
- **提交按钮**: 吸底全宽主按钮 "提交反馈"，明确告知用户此次反馈能帮助模型进阶。

---

## 🚀 3. 如何在 Figma 中使用
1. 使用 Chrome 浏览器打开当前代码库下的 `docs/ui_design.html`。
2. 在 Figma 社区安装并打开插件：[Figma to HTML, CSS, React & more!](https://www.figma.com/community/plugin/849822798253640248) (或类似 HTML to Figma 插件)。
3. 在插件界面按提示输入本地打开 `ui_design.html` 得到的 `localhost` 或 `file://` 地址。
4. 插件将自动解析该页面的 DOM、颜色、边距和文字属性，并在 Figma 工作区内生成高度可编辑的 Auto Layout 涂层与元素组。
