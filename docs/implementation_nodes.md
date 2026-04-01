# 实现节点记录

该文件按时间顺序记录每个功能实现节点。

| 节点 ID | 日期 | 版本 | 模块 | 功能说明 | 验证方式 | 提交 |
|---|---|---|---|---|---|---|
| N001 | 2026-02-21 | 0.1.0 | Monorepo 初始化 | 完成 MVP 架构骨架：API、推理服务、小程序、训练流水线、基础设施、CI。 | `make test` | `ac6d374` |
| N002 | 2026-02-21 | 0.1.1 | 运行稳定性 | 修复 API Docker 依赖复制问题，调整 compose 启动顺序确保 MySQL 就绪后启动 API。 | `make up && docker compose -f infra/docker-compose.yml ps` | `ffaa1e8` |
| N003 | 2026-02-21 | 0.2.0 | 训练与版本化 | 新增真实数据集训练脚本、ONNX 导出、推理先验加载、实现节点记录工具。 | `make test` + 训练/推理烟雾检查 | `TBD` |
| N004 | 2026-02-22 | 0.3.0 | 端侧运行时观测 | 用 `EdgeInferenceEngine` 替换固定 mock 调用；`finalize` 接入 `edgeRuntime` 入参和 `edgeMeta` 出参；保留端失败云兜底。 | `cd services/api && go test ./...` + `cd apps/wechat-miniprogram && npm run typecheck` | `f2c13d8` |
| N005 | 2026-02-22 | 0.4.0 | 数据流水线自动化 | 新增反馈清洗、加权清单构建、每日主动学习任务、续训参数支持、混淆矩阵产物。 | `cd ml/training && python3 -m unittest discover -s scripts -p 'test_*.py'` + `make test` | `1565346` |
| N006 | 2026-02-22 | 0.5.0 | 疑似不适提示 | 新增可选 pain-risk 分支、API 风险输出协议、文案免责声明强制注入、小程序风险卡片渲染（非诊断）。 | `make test` | `3349df5` |
| N007 | 2026-02-23 | 1.0.0 | 发布就绪 | 新增会话鉴权、限流与签名防护、模型发布管理接口、发布迁移、清理任务、v0.6 训练门禁脚本。 | `make test` | `904400c` |
| N008 | 2026-02-23 | 1.0.0 | 白名单配额 | 新增运行时白名单拦截与按用户每日配额控制，并更新配置与文档。 | `make test` | `3196cff` |
| N009 | 2026-02-23 | 1.0.0 | 灰度路由 | 实现按用户稳定分桶的灰度选择逻辑（确定性路由）。 | `make test` | `7d346e9` |
| N010 | 2026-02-23 | 1.0.0 | 客户端灰度接入 | 小程序消费 `selectedModel` 并接入白名单降级路径。 | `make test` | `ab125f4` |
| N011 | 2026-02-23 | 1.0.0 | 文档知识点 | 扩展项目手册知识点深度说明与覆盖检查清单。 | 人工审阅 + 命令引用校对 | `9993102` |
| N012 | 2026-02-24 | 1.0.0 | 文档重构 | 重建项目手册结构，合并重复信息并统一知识点章节。 | 人工审阅 + 章节关键字检查 | `4d04a21` |
| N013 | 2026-02-24 | 1.0.0 | docs-localization | 全仓文档统一切换为中文表达 | manual review | `cb22b40` |
| N014 | 2026-03-31 | 1.0.0 | 验收闭环 | 补齐本地烟雾链路、API 主链路集成测试和小程序最小页面测试，形成 M1 可回归验收基线。 | `make test` + `make smoke-local` | `4e290aa, 1462418, e7f7746` |
| N015 | 2026-03-31 | 1.0.0 | 云侧 ONNX | 完成 ONNX 接入决策、云侧双 predictor 结构、共享 uploads volume 和运行时/CI 修正，云侧推理可切到 ONNX 真推理。 | `cd services/inference && go test ./...` + `make test-go` + `docker compose -f infra/docker-compose.yml up -d --build` | `03aaa1d, 55d4f76, 2516d96, a365706, 01fcd72, 5a6a966, 6fd2e3b, 48a1229, 6c10b5a, f2b8951, 7590217` |
| N016 | 2026-03-31 | 1.0.0 | 最小可观测性 | 新增 API `/metrics` 端点，输出 finalize 延迟、错误率、fallback 比例与 copy 失败/超时等最小运行时指标，并补充上线指标文档。 | `cd services/api && go test ./...` + `curl http://127.0.0.1:8080/metrics` | `1e4083e, 53efb04` |
| N017 | 2026-04-01 | 1.0.0 | 观测告警基线 | 新增 Prometheus/Alertmanager 基础配置与告警规则，Compose 接入本地监控栈，`finalize_duration_ms_bucket` 补 `+Inf` 以支持 p95 告警查询。 | `make test-go` + `docker compose -f infra/docker-compose.yml config` | `72fcb3c` |
| N018 | 2026-04-01 | 1.0.0 | 训练自动调度 | 新增 `training_daily_pipeline.sh` 与 GitHub Actions 定时任务（每日 + 手动触发），缺少输入数据时自动跳过并保持任务成功。 | `make training-daily-pipeline` + `make test-py` | `e3f9597` |
| N019 | 2026-04-01 | 1.0.0 | 告警通知通道 | Alertmanager 新增 `critical` 级 webhook 路由，支持 `ALERT_WEBHOOK_URL` 环境变量注入，补充本地调试说明。 | `docker compose -f infra/docker-compose.yml config` | `0f9e57a` |
| N020 | 2026-04-01 | 1.0.0 | Grafana 预置看板 | Grafana 增加 provisioning 自动加载 Prometheus datasource 与运行时总览看板（错误率、p95、fallback、copy 超时）。 | `docker compose -f infra/docker-compose.yml config` + `python3 -m json.tool infra/monitoring/grafana/dashboards/meowlator-runtime-overview.json` | `1672d22` |
