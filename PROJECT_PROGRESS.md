# 矩龙项目进度记录（压缩快照）

最后更新：2026-10-17（Asia/Shanghai）
项目根目录：`/Users/xujs/.docker_dir/new-api`
技术登记入口：[`DEVELOPMENT.md`](./DEVELOPMENT.md)

## 维护约定

- 新对话或上下文恢复先读本文件当前快照、最新批次、未完成项和验证边界，再核对代码、`git status`、分支、远端和合并状态；根目录及相关子目录 `AGENTS.md` 仍必须读取。
- 本文件以后只维护“当前状态 + 未完成项 + 最新批次 + 必要验证 + Git/重启/部署结果”。旧批次压缩为一行索引，不无限追加完整历史；每批开始、里程碑、阻塞、交付和最终移交时原地更新。
- 只记录可核实事实。实现、测试、重启、提交、推送、生产部署分开写；Skip/未运行不能写成通过。禁止写入密钥、Cookie、session secret、密码、私密数据和大段 Base64。
- 业务 API、路由、模型、配置、迁移、组件和变更日志仍以 `DEVELOPMENT.md` 为详细登记入口；本文件只保留接续所需摘要。

## 当前状态

| 项目 | 当前事实 | 边界/下一步 |
| --- | --- | --- |
| Git | `main` / `HEAD 74f722cd3`；`origin/main` 同 SHA，远端为 `https://github.com/Xujs98/julong-api.git` | 本批文档压缩已提交并推送；只推 `origin`，不推 `upstream` |
| 上游 | `upstream = https://github.com/QuantumNous/new-api.git`；主分支已包含既有整合提交 | 不重复合并，不覆盖 Julong 实现 |
| 本地服务 | B24 曾以 PID 51076 验证监听 `3000`、`/api/status` 200；本次复核当前无监听 | 旧启动记录不等于当前运行；涉及后端改动时按规则重启并复核 |
| 既有未跟踪文件 | `controller/media_asset.go.bak`、`site-assets/`、`web/pnpm-lock.yaml`、`web/src/features/usage-logs/components/media-assets-table.tsx.bak` | 必须保留，不纳入无关提交 |
| 当前用户需求 | `PageSizeOptions` 尚未实现：系统设置→系统信息增加字符串配置，默认 `10,20,50,100`，统一所有表格“每页行数” | 下一功能批次实现后端选项、状态接口、系统设置权限、前端共享解析/分页下拉、i18n、测试和文档 |
| 生产 | 未部署线上 | 本地构建/启动不能写成生产发布 |

## 未完成项与限制

1. **PageSizeOptions（待实现）**：计划增加 `PageSizeOptions` 配置键、默认值与 1–100 正整数/去重校验；系统信息白名单和 `/api/status` 返回字段；前端表单、共享解析回退、所有分页下拉统一读取；保留输入顺序并补七语言、回归测试及 `DEVELOPMENT.md`。
2. **B04/B03 线上边界**：素材服务、成品直链和视频文档已本地验证；公网部署、Safari、原生音频直链和第三方端到端仍待环境具备后复核。
3. **B06**：Chromium 阴影/滚轮已验证，Safari 人工复核未完成。
4. **持续 QA**：全量 Go、真实管理员浏览器、ClickHouse 与生产验收不能由已有定向测试替代；按受影响范围补验证。

## 最新批次

### B24：启动本地项目（2026-10-17，历史已验证）

- 复用 `/tmp/julong-api-runtime/new-api-b23`，使用现有 `.env` 与 SQLite；未改业务代码或配置。
- 启动前 SQLite backup 权限 `0600`、`PRAGMA quick_check=ok`；PID 51076 曾监听 `3000`，localhost/127.0.0.1 的 `/api/status` 为 200、`success=true`，首页及 6 项资源与 `web/dist` 一致。
- 仅验证启动和静态资源；未重跑业务测试、三库矩阵、真实管理员浏览器或生产部署。当前复核服务已停止。

### B23：文本替换可视化编辑器与概览活跃人数卡片（已完成本地/已推送）

- 渠道和模型共享可视化/JSON 文本替换编辑器；概览活跃人数独立卡片；七语言同步。
- 前端测试 12/12、`tsgo -b`、目标 lint/format、i18n sync、Rsbuild 构建、Go 编译、重启和 `/api/status` 通过。
- 提交 `c0497299c3dc52df9d428b327db21baec1b1e601`，已推送并核对 `origin/main`；真实管理员浏览器和生产未验收。

### B22：渠道/模型请求与响应文本替换（已完成本地/已推送）

- 渠道默认与模型覆盖规则已接入多种 relay；支持 `request`/`response`、匹配模式、校验限制，保留原始数据供审计/usage/billing。
- Relay 定向 Go、relaykit 独立测试/构建、前端测试/typecheck/format/build、根 Go 构建、`git diff --check` 通过；本地服务曾以 PID 29600 激活。
- 提交 `f5d5be1cd`，已推送 `origin/main`；未部署生产。

### B21/B20：本地构建清理与 Bun 镜像修复（2026-10-17）

- 清理旧前端/Go 构建垃圾和测试镜像，保留源码、数据库、`web/dist`、`site-assets` 与既有未跟踪文件；B20 将 Docker 构建 Bun 固定到已验证的 1.4.1 digest。
- Docker build、前端构建、Go 编译和健康检查历史通过；提交链已包含在当前 `origin/main`。生产镜像未发布。

## 历史批次索引（压缩）

| 批次 | 主题 | 状态摘要 |
| --- | --- | --- |
| B01 | 原功能与上游整合基线 | 主分支已整合；持续保护 Julong 功能和 origin/upstream 边界 |
| B02 | 后台页面、密钥滚动、自助发票 | 历史实现；后续 schema 变更仍需三库验证 |
| B03 | 算力视频渠道、任务日志、成品播放 | 本地接口/Range/Chromium 播放验证；线上与第三方端到端待验 |
| B04 | 输入素材托管与本站视频文档 | 三库、回归、浏览器本地验证完成；公网部署边界待验 |
| B05 | 模型广场视频按秒/按次价格 | 已实现；价格显示规则有历史验证 |
| B06 | 成品列阴影与滚轮 | 本地/Chromium 已验证；Safari 待人工复核 |
| B07 | 项目进度入口 | 本文与 AGENTS.md 建立强制接续规则 |
| B08 | Markdown/Word 模型文档模板 | CRUD、绑定、展示与安全修复完成；定向验证，生产未部署 |
| B09 | 素材管理 UI、筛选、重建 | 本地验证并交付；旧无关侧栏失败不扩大为全量通过 |
| B10 | 任务日志视频下载文件名 | 本地验证并交付 |
| B11 | 模型 API 使用开关 | 后端/前端/三库/构建验证；提交链已交付 |
| B12 | 本地构建缓存清理 | 已完成，保留运行资源和源码 |
| B13 | 客服群二维码与顶部入口 | 前端、i18n、构建和本地激活已验证 |
| B14 | 管理员任务日志权限与 Root 隐私 | 跨账户权限、Root 隐私、三库/前端/构建/重启已验证 |
| B15 | 禁用账户全站拦截与自动追封 | 定向路由/三库/登录回归通过；人机验证与生产待验 |
| B16 | 使用日志权限与用户在线统计 | 双来源、权限、三库、前端、构建/重启已验证 |
| B17 | 网站操作与 API 在线分离 | 本地完成并推送；全量 Go、真实浏览器、生产未验收 |
| B18/B19 | 清理 IPv4/IPv6 本机封禁 | SQLite 备份、删除、quick_check、重启和本地健康通过 |

### B25：压缩开发规则与项目进度（2026-10-17，已完成并推送）

- `AGENTS.md` 从 214 行压缩为 63 行，保留进度门禁、三数据库、认证、计费、插件、前端、i18n、测试、Git、保护标识和 issue/PR 规则；`PROJECT_PROGRESS.md` 从 481 行压缩为 99 行，旧批次改为索引。
- `DEVELOPMENT.md` 仅同步压缩快照维护规则；四项既有未跟踪文件保留，未改业务代码、API、数据库、前端资源或本地服务。
- `git diff --check` 通过。提交 `74f722cd3` 已推送 `origin/main`，`git ls-remote` 已核对同一 SHA；生产未部署。

## 验证口径

- 已有三库记录：SQLite 3.50.4、MySQL 5.7.44、PostgreSQL 16.15；涉及 schema/数据库行为时必须按 `AGENTS.md` 重新运行真实 fresh/upgrade/idempotency 矩阵。
- 已有前端记录包含 typecheck、目标 lint/format、i18n sync、Vitest/组件回归和 Rsbuild build；不得把历史结果包装成当前代码已重新验证。
- 已有后端记录包含定向 Go 测试、根构建、relaykit 独立构建、重启、`/api/status` 和资源 hash；全量 Go、真实管理员浏览器、ClickHouse、Safari、公网和生产仍按上表保留边界。
- 本次文档压缩只改文档规则与进度摘要，不改业务代码、API、数据库或前端资源；无需重启服务。备份位于 `/tmp/julong-doc-compress-20261017/`。

## 后续批次记录格式（必须压缩）

### Bxx：任务名称（YYYY-MM-DD，状态）

- 用户目标：
- 完成/未完成：
- 文件/API/路由/配置：
- 实际验证：命令、结果、失败/Skip、版本和局限；未运行不写通过。
- 重启/健康：PID、端口、端点和资源核对；未重启写明原因。
- Git/部署：commit、remote/branch、push 结果、生产状态。
- 下一步：只写仍需执行的动作。

当前:文档压缩 / 规则和进度已压缩，B25 已提交并推送 origin/main，业务功能未改 / 下一步按压缩模板记录并实现 PageSizeOptions
