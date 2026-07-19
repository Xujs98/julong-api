# Julong-API 开发文档

最后更新：2026-07-19

本文档是本二开项目的强制开发记录。以后新增、修改或删除任何 API、组件、数据模型、配置项、路由、数据库行为或部署行为时，必须在同一次改动中同步更新本文档，并在“变更日志”中新增记录。

## 维护规则

- 当前代码是唯一事实来源。如果本文档和代码不一致，必须先按当前代码修正文档，再继续开发。
- 每次 API 变更必须同步更新 API 清单、请求/响应说明、权限说明和变更日志。
- 每次前端组件、页面、交互变更必须同步更新组件清单或对应功能模块说明。
- 每次数据库模型、字段或迁移行为变更必须同步更新数据模型和迁移注意事项。
- 前端 i18n 语言包禁止手动直接编辑。新增或修改翻译必须通过临时脚本 `web/default/scripts/add-missing-keys.mjs` 写入，然后运行 `bun run i18n:sync`，最后删除临时脚本。
- `web/default/src/routeTree.gen.ts` 是 TanStack Router 工具链/typecheck 生成文件。除非生成不可用且改动完全机械明确，否则不要手动编辑。
- 当前项目名为 Julong-API / 矩龙-API。除非保留上游 import path、包名或兼容代码确有必要，不要新增面向用户的 `new-api` 文案。

## 项目状态

| 模块 | 状态 | 说明 |
| --- | --- | --- |
| 后端 Gin API | 进行中 | 保留上游核心功能，并叠加 Julong 二开功能。本地后端通常监听 `:3000`。 |
| 默认前端 UI | 进行中 | React 19 + Rsbuild + TanStack Router，代码在 `web/default`。本地开发服务通常是 `:5173`。 |
| Classic 前端 | 上游已有 | Dockerfile 仍会构建，但不是当前主要二开目标。 |
| Docker 部署 | 进行中 | `docker-compose.yml` 使用 `qq1371446705/julong-api:latest`，包含 Postgres、Redis，宿主端口 `3388`。 |
| 代理功能 | 已实现，仍需持续 QA | 代理折扣、代理生成兑换码、代理所属用户、代理充值链接、退款日志等。 |
| 错误反馈工单 | 已实现 | 500 页面跳转 `/error-report`，管理员/root 在 `/error-reports` 查看。 |
| 用户详情与 IP 管理 | 已实现 | 后台用户详情显示登录 IP 历史并支持多选封禁/解封；用户列表标记共享 IP 和已封 IP。 |
| 兑换码搜索 | 已实现 | 后台兑换码支持按兑换码 key、生成者用户名/显示名、名称、ID、状态搜索。 |
| 签到额度预览 | 已实现 | 计费与支付中的签到奖励输入框显示格式化额度预览。 |
| 生图日志与异步生图 | 已实现 | 开启生图日志后，`async: true` 立即创建 `pending` 日志并返回任务 ID，支持 API Key 轮询、状态更新、图片读取和后台 JSON 详情；关闭日志时退回同步且不存图。 |
| 系统设置权限 | 已实现 | root 可按 7 个一级菜单下的 41 个二级设置页面逐项授权管理员；菜单、路由、通用配置键和专用 API 均执行权限校验。 |
| 客服联系 | 已实现 | 概览页展示联系客服弹窗；root 可授权指定管理员维护多条 QQ、微信和手机联系方式。 |
| 文档纪律 | 新增 | 以后每次代码改动都必须同步维护本文档。 |

## 仓库结构

| 路径 | 用途 | 关键依赖 | 状态 |
| --- | --- | --- | --- |
| `main.go` | 进程入口、嵌入前端资源、初始化路由和后台任务 | `router`、`model`、`common`、`service` | 活跃 |
| `router/` | Gin 路由注册：管理 API、Relay API、Dashboard、Video、前端静态资源 | `controller`、`middleware`、`relay` | 活跃 |
| `controller/` | HTTP handler、参数校验、响应封装 | `model`、`service`、`common`、`setting` | 活跃 |
| `model/` | GORM 模型、数据库迁移、持久化辅助函数 | `gorm`、`common`、`setting` | 活跃 |
| `middleware/` | 鉴权、CORS、限流、请求 ID、请求体清理、审计 | `common`、`model`、`service/authz` | 活跃 |
| `service/` | 业务逻辑：Relay、计费、订阅、任务、OAuth、token 计数 | `relay`、`model`、各 provider SDK | 活跃 |
| `relay/` | OpenAI/Anthropic/Gemini/MJ/task 兼容网关逻辑 | channel adapters、billing helpers | 活跃 |
| `setting/` | 运行时配置分组和默认设置 | `model.Option`、环境变量 | 活跃 |
| `common/` | 公共工具、数据库初始化、额度计算、分页、API 响应工具 | stdlib、Redis、GORM | 活跃 |
| `web/default/` | 主 React 前端 | React、Rsbuild、TanStack Router/Query/Table、Tailwind、shadcn 风格组件 | 活跃 |
| `web/classic/` | Classic 主题前端 | Vite/React 旧栈 | 上游已有 |
| `docker-compose.yml` | 生产风格 Compose 部署 | Postgres、Redis、镜像 `qq1371446705/julong-api:latest` | 活跃 |
| `docker-compose.dev.yml` | 本地后端构建的开发 Compose | Postgres、Redis、Dockerfile.dev | 需要品牌名清理 |
| `Dockerfile` | 多阶段构建 default/classic 前端和 Go 二进制 | Bun、Go、Debian runtime | 活跃 |

## 技术栈

### 后端

- 语言：Go，模块名仍为 `github.com/QuantumNous/new-api`，`go 1.25.1`。
- HTTP 框架：Gin。
- ORM：GORM。
- 数据库：
  - 主库：默认 SQLite；通过 `SQL_DSN` 可使用 MySQL/PostgreSQL。
  - 日志库：默认跟随主库；可通过 `LOG_SQL_DSN` 使用独立数据库或 ClickHouse。
  - Redis：可选，通过 `REDIS_CONN_STRING` 启用。
- 鉴权：
  - Dashboard 使用 session 鉴权。
  - 管理 API 和部分接口支持 access token 鉴权。
  - `/v1`、`/mj`、task/video relay 路由使用 API key token 鉴权。
  - 角色常量来自 `common`：普通用户、管理员、root。
- 计费：
  - 额度以整数 quota 单位存储。
  - 前端显示通过 `web/default/src/lib/format.ts` 和货币显示设置格式化。
  - 余额和订阅额度并存；按次计费模型可以配置是否允许订阅额度抵扣。

### 前端

- 工作区：`web/`。
- 主应用：`web/default`。
- 包管理/运行时：Bun。本机已知 Bun 路径：`/Users/xujs/.nvm/versions/node/v24.13.0/bin/bun`。
- 框架：React 19。
- 路由：TanStack Router，文件路由位于 `web/default/src/routes`。
- 数据请求：TanStack Query + Axios 封装 `web/default/src/lib/api.ts`。
- 表格：TanStack Table + 共享表格系统 `web/default/src/components/data-table`。
- 样式：Tailwind CSS 4、Base UI、本地 shadcn 风格 primitives。
- i18n：`react-i18next`，语言文件位于 `web/default/src/i18n/locales`。

## 本地开发

### 后端

```bash
go run main.go
```

默认本地行为：

- HTTP 监听 `http://localhost:3000`。
- 未设置 `SQL_DSN` 时使用 SQLite 文件 `one-api.db`。
- 未设置 `REDIS_CONN_STRING` 时 Redis 不启用。
- 启动时会对 `model/main.go` 中列出的模型执行 AutoMigrate。

### 前端

```bash
cd web/default
PATH="/Users/xujs/.nvm/versions/node/v24.13.0/bin:$PATH" \
  /Users/xujs/.nvm/versions/node/v24.13.0/bin/bun run dev
```

常用开发地址：`http://localhost:5173`。

### 验证命令

```bash
go test ./...
cd web/default
PATH="/Users/xujs/.nvm/versions/node/v24.13.0/bin:$PATH" bun run typecheck
PATH="/Users/xujs/.nvm/versions/node/v24.13.0/bin:$PATH" bun run i18n:sync
git diff --check
```

## 部署

### Docker Compose

当前生产风格 Compose 文件：`docker-compose.yml`。

关键值：

- 项目名：`julong-api`。
- 应用镜像：`qq1371446705/julong-api:latest`。
- 应用容器：`julong-api`。
- 宿主端口：`3388:3000`。
- Redis 容器：`julong-api-redis`。
- Postgres 容器：`julong-api-postgres`。
- 主库 DSN 示例：`postgresql://root:123456@postgres:5432/julong-api`。
- 数据卷：`./data:/data`、`./logs:/app/logs`、`pg_data`。

生产部署前必须修改 Compose 中所有默认密码。

推荐服务器更新流程：

```bash
git pull
docker compose pull
docker compose up -d
```

该流程会在镜像或 Compose 变化时重新创建容器。只要不执行 `docker compose down -v` 或手动删除 volume，命名数据库卷中的数据会保留。

## 运行时配置

配置通过 `model.Option` 持久化，由 `controller/option.go` 和前端系统设置页面管理。

| Key 或分组 | 文件 | 用途 | 备注 |
| --- | --- | --- | --- |
| `checkin_setting.enabled` | `setting/operation_setting/checkin_setting.go`、`web/default/src/features/system-settings/general/checkin-settings-section.tsx` | 启用每日签到奖励 | 前端已显示格式化额度预览。 |
| `checkin_setting.min_quota` | 同上 | 随机签到奖励最小额度 | 存储原始 quota 整数。 |
| `checkin_setting.max_quota` | 同上 | 随机签到奖励最大额度 | 存储原始 quota 整数。 |
| `QuotaForNewUser` | `quota-settings-section.tsx` | 新用户初始额度 | 使用 `formatQuota` 显示预览。 |
| `QuotaForInviter` / `QuotaForInvitee` | 同上 | 邀请奖励 | 受支付合规确认约束。 |
| `SidebarModulesAdmin` | `web/default/src/features/system-settings/maintenance/config.ts` | 控制侧边栏模块显示 | 自定义模块包含 `errorReports`。 |
| 模型定价订阅抵扣 | `web/default/src/features/system-settings/models/model-pricing-sheet.tsx` 及后端计费设置 | 按次模型是否允许订阅额度抵扣 | 适用于所有模型，不局限于 `gpt-image-2`。 |
| `ImageGenerationLogEnabled` | `common/constants.go`、`model/option.go`、`log-settings-section.tsx` | 是否记录生图请求并启用本地异步生图 | Root 或获授权管理员配置，默认 `false`；关闭时 `async: true` 退回同步，不创建任务、不捕获或保存图片。 |
| `ImageGenerationLogRetentionDays` | 同上、`service/image_generation_log.go` | 生图日志及本地图片自动保留天数 | 默认 `30`，范围 `0-3650`；`0` 表示永久保留，每小时至多触发一次过期清理。 |
| `ImageGenerationLogPollingIntervalSeconds` | `common/constants.go`、`model/option.go`、`log-settings-section.tsx` | 未完成生图任务的前端轮询频率 | 默认 `15` 秒，允许 `5-3600` 秒；同时通过 `/api/status` 和任务响应公开。 |
| `ImageGenerationLogImageAuthWhitelistEnabled` | `common/constants.go`、`model/option.go`、`middleware/image_generation_image_auth.go`、`log-settings-section.tsx` | 是否允许图片读取白名单免 API Key | 默认 `false`；只作用于 `GET /v1/images/generations/:task_id/images/:index`，不放开任务 JSON 查询和生图提交。 |
| `ImageGenerationLogImageAuthWhitelist` | 同上、`common/image_generation_image_auth_whitelist.go` | 图片读取免鉴权来源列表 | 换行/逗号分隔精确 IP 或域名；IP 匹配 `ClientIP`，域名匹配浏览器 `Origin/Referer`；URL 会归一化为主机名；不支持通配符/CIDR；`0.0.0.0` 表示全部放行。 |
| `IMAGE_LOG_STORAGE_DIR` | `service/image_generation_log.go` | 覆盖生图图片文件目录 | 默认 `image-generation-logs`；Docker `WORKDIR /data` 下位于持久化卷 `/data/image-generation-logs`。 |
| `ImageGenerationStorage*` | `service/image_object_storage.go`、`controller/image_object_storage.go`、`log-settings-section.tsx` | 异步生图 MinIO/S3 私有对象存储 | 在“系统设置 → 运维 → 日志维护 → 记录生图日志”中配置；包含启用、Endpoint、Bucket、Region、Access/Secret Key、HTTPS、Path Style 和对象前缀。Secret Key 不回传前端，留空保存/测试沿用已存密钥。默认 Bucket `julong-media`、前缀 `generated/images`。 |
| `async` 生图请求参数 | `dto/openai_image.go`、`controller/image_generation_task.go` | 将同步 `/v1/images/generations` 包装为本地异步任务 | 默认 `false`；仅在 `ImageGenerationLogEnabled=true` 时生效；转发上游前删除；有效异步模式与 `stream: true` 互斥。 |
| `SupportContacts` | `common/constants.go`、`model/option.go` | 客服联系方式 JSON 数组 | 每项包含 `type`（qq/wechat/phone）、`label`、`value`；最多 30 条。 |

## 架构

### 请求流程

1. 客户端访问前端路由或 API。
2. 前端 Axios 实例 `web/default/src/lib/api.ts` 在可用时从 localStorage 附加 `New-Api-User` 请求头。
3. Gin 路由组应用中间件：
   - `middleware.UserAuth()`：登录用户。
   - `middleware.AdminAuth()`：管理员/root。
   - `middleware.RootAuth()`：仅 root。
   - `middleware.TokenAuth()`：Relay API key。
4. Controller 校验请求并调用 model/service 层。
5. Dashboard API 通常返回统一业务响应：

```json
{
  "success": true,
  "message": "",
  "data": {}
}
```

Relay API 使用 OpenAI/Anthropic/Gemini 兼容响应和错误结构。

### 前端流程

1. `web/default/src/routes` 下的文件路由渲染 feature 组件。
2. Feature 的 `api.ts` 调用 `web/default/src/lib/api.ts`。
3. Feature 表格使用 `DataTablePage` 和 `useTableUrlState` 管理分页、过滤、URL 状态、移动端/桌面端布局。
4. Mutation 通常使用 `sonner` toast，并触发 provider 中的 refresh 状态。
5. 管理页面通过 route `beforeLoad` 读取 `useAuthStore` 和 `ROLE` 常量做权限守卫。

## API 清单

除非特别说明，Dashboard API 返回结构如下：

```ts
type ApiResponse<T> = {
  success: boolean
  message?: string
  data?: T
}
```

通用错误处理：

- 鉴权中间件可能因 session/access token 缺失或无效返回 HTTP `401`。
- 业务校验错误通常返回 HTTP `200` 且 `success:false`。
- Relay 未找到或不支持的路由返回 OpenAI 兼容错误 JSON。
- 前端 Axios 默认对 `success:false` 和 HTTP 错误弹 toast，除非请求配置关闭错误处理。

### 公共与系统 API

| 方法 | 路径 | Handler | 用途 | 参数/请求体 | 响应 | 权限 | 状态 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| GET | `/api/setup` | `controller.GetSetup` | 读取安装/初始化状态 | 无 | setup 元数据 | 公开 | 完成 |
| POST | `/api/setup` | `controller.PostSetup` | 初始化安装 | setup payload | success | 公开 + body limit | 完成 |
| GET | `/api/status` | `controller.GetStatus` | 系统状态/健康检查 | 无 | status 对象；包含 `custom_endpoints`、`image_generation_log_polling_interval_seconds`；公告启用时仅含当前有效且面向所有用户的 `announcements`，并返回 `announcements_enabled` | 公开 | 完成 |
| GET | `/api/announcements` | `controller.GetUserAnnouncements` | 获取当前用户可见公告 | 无 | 已展示、时间有效且命中套餐/余额 OR-AND 条件的 `Announcement[]` | 登录用户 | 完成 |
| GET | `/api/uptime/status` | `controller.GetUptimeKumaStatus` | Uptime Kuma 集成 | 无 | uptime 状态 | 公开 | 完成 |
| GET | `/api/notice` | `controller.GetNotice` | 站点公告 | 无 | 内容 | 公开 | 完成 |
| GET | `/api/user-agreement` | `controller.GetUserAgreement` | 用户协议 | 无 | markdown/html | 公开 | 完成 |
| GET | `/api/privacy-policy` | `controller.GetPrivacyPolicy` | 隐私政策 | 无 | markdown/html | 公开 | 完成 |
| GET | `/api/about` | `controller.GetAbout` | 关于页面内容 | 无 | 内容 | 公开 | 完成 |
| GET | `/api/home_page_content` | `controller.GetHomePageContent` | 首页内容 | 无 | JSON 内容 | 公开 | 完成 |
| GET | `/api/pricing` | `controller.GetPricing` | 定价/模型广场数据 | query filters | pricing data | 顶部导航模块权限 | 完成 |
| GET | `/api/rankings` | `controller.GetRankings` | 排行榜数据 | query filters | ranking data | 顶部导航模块权限 | 完成 |
| GET | `/api/ratio_config` | `controller.GetRatioConfig` | 暴露给前端的倍率配置 | 无 | ratio object | Critical rate limit | 完成 |
| GET | `/api/perf-metrics/summary` | `controller.GetPerfMetricsSummary` | 性能指标摘要 | query | summary | pricing 导航 public/user auth | 完成 |
| GET | `/api/perf-metrics` | `controller.GetPerfMetrics` | 性能指标列表 | query | list | pricing 导航 public/user auth | 完成 |

`GET /api/announcements` 无请求体，要求登录 Session 或用户 access token，并携带现有鉴权中间件要求的 `New-Api-User`。成功响应为 `{success:true,data:Announcement[]}`；后端只返回 `status=active`、处于起止时间内且命中当前用户套餐/余额条件的公告。未登录返回 HTTP 401，用户或订阅查询失败返回标准 API 错误。调用示例：

```bash
curl 'http://localhost:3000/api/announcements' \
  -H 'New-Api-User: 1' \
  -H 'Authorization: Bearer <user-access-token>'
```

### 错误反馈 API

| 方法 | 路径 | Handler | 用途 | 参数/请求体 | 响应 | 权限 | 状态 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| POST | `/api/error-reports` | `controller.SubmitErrorReport` | 从 500 页面提交错误反馈 | JSON `{title,message,page_url,error_status,stack?}` | `ErrorReport` | 公开；通过 `TryUserAuth` 可选记录登录用户；匿名 body limit | 完成 |
| GET | `/api/error-reports` | `controller.GetErrorReports` | 管理员错误反馈列表 | `p`, `page_size` | `PageInfo<ErrorReport[]>` | 管理员/root | 完成 |
| GET | `/api/error-reports/` | `controller.GetErrorReports` | 同上，兼容尾斜杠 | `p`, `page_size` | `PageInfo<ErrorReport[]>` | 管理员/root | 完成 |
| GET | `/api/error-reports/:id` | `controller.GetErrorReport` | 查看单条错误反馈 | path `id` | `ErrorReport` | 管理员/root | 完成 |

调用示例：

```bash
curl http://localhost:3000/api/error-reports \
  -H 'Content-Type: application/json' \
  -d '{"title":"500 on model pricing","message":"Clicked model pricing and got 500","page_url":"http://localhost:5173/system-settings/billing/model-pricing","error_status":500}'
```

### 认证与用户 API

| 方法 | 路径 | Handler | 用途 | 参数/请求体 | 权限 | 状态 |
| --- | --- | --- | --- | --- | --- | --- |
| POST | `/api/user/register` | `controller.Register` | 注册用户 | username/password/email 等 | 公开 + Turnstile/rate limit | 完成 |
| POST | `/api/user/login` | `controller.Login` | 登录 | credentials | 公开 + Turnstile/rate limit | 完成 |
| POST | `/api/user/login/2fa` | `controller.Verify2FALogin` | 完成 2FA 登录 | 2FA code | 公开 + rate limit | 完成 |
| GET | `/api/user/logout` | `controller.Logout` | 登出当前 session | 无 | 公开 | 完成 |
| GET | `/api/user/groups` | `controller.GetUserGroups` | 用户分组元数据 | 无 | 公开 | 完成 |
| GET | `/api/user/self` | `controller.GetSelf` | 当前用户信息 | 无 | 用户 | 完成 |
| PUT | `/api/user/self` | `controller.UpdateSelf` | 更新当前用户资料 | user fields | 用户 + critical rate limit | 完成 |
| DELETE | `/api/user/self` | `controller.DeleteSelf` | 删除当前用户 | 无 | 用户 | 完成 |
| GET | `/api/user/token` | `controller.GenerateAccessToken` | 生成管理/用户 access token | 无 | 用户 | 完成 |
| GET | `/api/user/models` | `controller.GetUserModels` | 用户可用模型 | 无 | 用户 | 完成 |
| GET | `/api/user/aff` | `controller.GetAffCode` | 邀请码 | 无 | 用户 | 完成 |
| POST | `/api/user/aff_transfer` | `controller.TransferAffQuota` | 邀请额度转余额 | `{quota}` | 用户 + 合规确认 | 完成 |
| PUT | `/api/user/setting` | `controller.UpdateUserSetting` | 更新用户设置 JSON | setting payload | 用户 | 完成 |
| GET | `/api/user/agent/users` | `controller.GetAgentUsers` | 代理所属用户 | 无 | 代理用户 | 完成 |
| PUT | `/api/user/agent/topup-link` | `controller.UpdateAgentTopUpLink` | 代理自定义充值链接 | `{agent_topup_link}` | 代理用户 | 完成 |
| GET | `/api/user/checkin` | `controller.GetCheckinStatus` | 读取每日签到状态 | 无 | 用户 | 完成 |
| POST | `/api/user/checkin` | `controller.DoCheckin` | 执行每日签到 | 无 | 用户 + Turnstile | 完成 |

### 后台用户 API

以下路由均位于 `/api/user` 下，除非特别说明，均需要 `middleware.AdminAuth()`。

| 方法 | 路径 | Handler | 用途 | 参数/请求体 | 响应 | 状态 |
| --- | --- | --- | --- | --- | --- | --- |
| GET | `/api/user/` | `controller.GetAllUsers` | 分页用户列表 | `p`, `page_size` | `PageInfo<User[]>` | 完成 |
| GET | `/api/user/search` | `controller.SearchUsers` | 搜索用户 | `keyword`, `group`, `role`, `status`, `p`, `page_size` | `PageInfo<User[]>` | 完成 |
| GET | `/api/user/:id` | `controller.GetUser` | 可编辑用户详情 | path `id` | `User`（包含 `last_login_at`、`last_login_ip`） | 完成；管理员/root，同级/更高角色受限 |
| POST | `/api/user/` | `controller.CreateUser` | 创建用户 | `UserFormData` | `User` | 完成 |
| PUT | `/api/user/` | `controller.UpdateUser` | 更新用户 | `UserFormData & {id}` | partial `User` | 完成 |
| DELETE | `/api/user/:id` | `controller.DeleteUser` | 删除用户 | path `id` | success | 完成 |
| POST | `/api/user/manage` | `controller.ManageUser` | 晋升/降级/启用/禁用/删除/额度调整；`disable` 同时封禁全部已知登录 IP | `{id,action,...}` | partial `User` | 完成 |
| GET | `/api/user/agent-detail/:id` | `controller.AdminGetAgentDetail` | 代理详情弹窗 | path `id` | `{agent,users,redemptions}` | 完成 |
| GET | `/api/user/:id/usage-summary` | `controller.AdminGetUserUsageSummary` | 用户详情弹窗总 token 消耗 | path `id` | `{total_tokens:number}` | 完成 |
| GET | `/api/user/:id/login-ips` | `controller.AdminGetUserLoginIPs` | 用户历史登录 IP、登录次数、最后使用时间、封禁和共享状态 | path `id` | `UserLoginIPStat[]` | 完成；管理员仅可查看低权限用户，root 可查看全部 |
| PUT | `/api/user/:id/login-ips` | `controller.AdminUpdateUserLoginIPs` | 批量封禁或解封登录 IP | path `id`；`{ips:string[],blocked:boolean}`，1-100 个标准 IPv4/IPv6 | success | 完成；管理员仅可操作低权限用户，root 可操作全部；非法 IP 返回参数错误 |
| GET | `/api/user/topup` | `controller.GetAllTopUps` | 管理员充值列表 | `p`, `page_size` | top-up page | 完成 |
| POST | `/api/user/topup/complete` | `controller.AdminCompleteTopUp` | 手动完成充值订单 | `{trade_no}` | success | 完成 |
| GET | `/api/user/:id/oauth/bindings` | `controller.GetUserOAuthBindingsByAdmin` | OAuth 绑定列表 | path `id` | binding list | 完成 |
| DELETE | `/api/user/:id/oauth/bindings/:provider_id` | `controller.UnbindCustomOAuthByAdmin` | 删除自定义 OAuth 绑定 | path params | success | 完成 |
| DELETE | `/api/user/:id/bindings/:binding_type` | `controller.AdminClearUserBinding` | 清除内置绑定 | path params | success | 完成 |
| DELETE | `/api/user/:id/reset_passkey` | `controller.AdminResetPasskey` | 重置 Passkey | path `id` | success | 完成 |
| GET | `/api/user/2fa/stats` | `controller.Admin2FAStats` | 2FA 统计 | 无 | stats | 完成 |
| DELETE | `/api/user/:id/2fa` | `controller.AdminDisable2FA` | 禁用用户 2FA | path `id` | success | 完成 |

IP 管理调用示例：

```bash
curl 'http://localhost:3000/api/user/12/login-ips' \
  -H 'Cookie: <admin-session-cookie>' \
  -H 'New-Api-User: <admin-user-id>'

curl -X PUT 'http://localhost:3000/api/user/12/login-ips' \
  -H 'Content-Type: application/json' \
  -H 'Cookie: <admin-session-cookie>' \
  -H 'New-Api-User: <admin-user-id>' \
  -d '{"ips":["203.0.113.8","2001:db8::8"],"blocked":true}'
```

错误处理：空列表、超过 100 个 IP 或任一非法 IP 返回参数错误且不写入；目标用户不存在返回查询错误；管理员操作同级/更高角色返回权限错误；主库或日志库异常返回数据库错误。写接口经过管理员审计中间件，并由 `recordManageAuditFor` 记录目标用户、操作数量和封禁状态。

调用示例：

```bash
curl 'http://localhost:3000/api/user/1/usage-summary' \
  -H 'Cookie: <admin-session-cookie>' \
  -H 'New-Api-User: 1'
```

### 支付、钱包、订阅 API

| 方法 | 路径 | Handler | 用途 | 权限 | 状态 |
| --- | --- | --- | --- | --- | --- |
| GET | `/api/user/topup/info` | `controller.GetTopUpInfo` | 钱包充值配置 | 用户 | 完成 |
| GET | `/api/user/topup/self` | `controller.GetUserTopUps` | 当前用户充值记录 | 用户 | 完成 |
| POST | `/api/user/topup` | `controller.TopUp` | 兑换码/充值动作 | 用户 | 完成 |
| POST | `/api/user/pay` | `controller.RequestEpay` | EPay 支付请求 | 用户 | 完成 |
| POST | `/api/user/amount` | `controller.RequestAmount` | 计算应付金额 | 用户 | 完成 |
| POST | `/api/user/stripe/pay` | `controller.RequestStripePay` | Stripe 支付 | 用户 | 完成 |
| POST | `/api/user/stripe/amount` | `controller.RequestStripeAmount` | Stripe 金额计算 | 用户 | 完成 |
| POST | `/api/user/creem/pay` | `controller.RequestCreemPay` | Creem 支付 | 用户 | 完成 |
| POST | `/api/user/waffo/amount` | `controller.RequestWaffoAmount` | Waffo 金额计算 | 用户 | 完成 |
| POST | `/api/user/waffo/pay` | `controller.RequestWaffoPay` | Waffo 支付 | 用户 | 完成 |
| POST | `/api/user/waffo-pancake/amount` | `controller.RequestWaffoPancakeAmount` | Waffo Pancake 金额计算 | 用户 | 完成 |
| POST | `/api/user/waffo-pancake/pay` | `controller.RequestWaffoPancakePay` | Waffo Pancake 支付 | 用户 | 完成 |
| POST | `/api/stripe/webhook` | `controller.StripeWebhook` | Stripe 回调 | 公开 body limit | 完成 |
| POST | `/api/creem/webhook` | `controller.CreemWebhook` | Creem 回调 | 公开 body limit | 完成 |
| POST | `/api/waffo/webhook` | `controller.WaffoWebhook` | Waffo 回调 | 公开 body limit | 完成 |
| POST | `/api/waffo-pancake/webhook/:env` | `controller.WaffoPancakeWebhook` | Waffo Pancake 回调 | 公开 body limit | 完成 |
| GET/POST | `/api/user/epay/notify` | `controller.EpayNotify` | EPay 通知 | 公开 | 完成 |
| GET/POST | `/api/subscription/epay/notify` | `controller.SubscriptionEpayNotify` | 订阅 EPay 通知 | 公开 | 完成 |
| GET/POST | `/api/subscription/epay/return` | `controller.SubscriptionEpayReturn` | 订阅 EPay 返回 | 公开 | 完成 |

`/api/subscription` 下用户订阅路由需要 `UserAuth()`：

- `GET /plans`
- `GET /self`
- `PUT /self/preference`
- `POST /balance/pay`
- `POST /epay/pay`
- `POST /stripe/pay`
- `POST /creem/pay`
- `POST /waffo-pancake/pay`

`/api/subscription/admin` 下订阅管理路由需要 `AdminAuth()`：

- `GET/POST/PUT/PATCH /plans`
- `POST /bind`
- `POST /plans/:id/subscriptions/reset`
- `GET/POST /users/:id/subscriptions`
- `POST /users/:id/subscriptions/reset`
- `POST /user_subscriptions/:id/invalidate`
- `DELETE /user_subscriptions/:id`

### Token/API Key API

所有 `/api/token` 路由都需要 `UserAuth()`。

| 方法 | 路径 | Handler | 用途 | 状态 |
| --- | --- | --- | --- | --- |
| GET | `/api/token/` | `controller.GetAllTokens` | 用户 API key 列表 | 完成 |
| GET | `/api/token/search` | `controller.SearchTokens` | 搜索 API key | 完成 |
| GET | `/api/token/:id` | `controller.GetToken` | API key 详情 | 完成 |
| POST | `/api/token/:id/key` | `controller.GetTokenKey` | 显示 key 明文 | 完成 |
| POST | `/api/token/` | `controller.AddToken` | 创建 API key | 完成 |
| PUT | `/api/token/` | `controller.UpdateToken` | 更新 API key | 完成 |
| DELETE | `/api/token/:id` | `controller.DeleteToken` | 删除 API key | 完成 |
| POST | `/api/token/batch` | `controller.DeleteTokenBatch` | 批量删除 key | 完成 |
| POST | `/api/token/batch/keys` | `controller.GetTokenKeysBatch` | 批量显示 key | 完成 |
| GET | `/api/token/:id/usage/` | `controller.GetTokenUsage` | Token 使用统计 | 完成 |

### 兑换码 API

`/api/redemption` 路由需要 `UserAuth()`。管理员/root 可查看全部；代理用户通过 `getRedemptionScopeUserId` 仅能访问自己生成的兑换码。

| 方法 | 路径 | Handler | 用途 | 参数/请求体 | 响应 | 状态 |
| --- | --- | --- | --- | --- | --- | --- |
| GET | `/api/redemption/` | `controller.GetAllRedemptions` | 兑换码列表 | `p`, `page_size` | `PageInfo<Redemption[]>` | 完成 |
| GET | `/api/redemption/search` | `controller.SearchRedemptions` | 搜索兑换码 | `keyword`, `status`, `p`, `page_size` | `PageInfo<Redemption[]>` | 完成；搜索 ID、名称、key、生成者用户名/显示名 |
| GET | `/api/redemption/:id` | `controller.GetRedemption` | 兑换码详情 | path `id` | `Redemption` | 完成 |
| POST | `/api/redemption/` | `controller.AddRedemption` | 创建兑换码 | `{name,quota,count,expired_time}` | generated keys | 完成；代理扣费规则生效 |
| PUT | `/api/redemption/` | `controller.UpdateRedemption` | 更新兑换码/状态 | `Redemption` payload | `Redemption` | 完成 |
| DELETE | `/api/redemption/invalid` | `controller.DeleteInvalidRedemption` | 删除无效/已用/过期兑换码 | 无 | count | 完成 |
| DELETE | `/api/redemption/:id` | `controller.DeleteRedemption` | 删除单个兑换码 | path `id` | success | 完成；未使用的代理兑换码退还 `agent_charge` |

代理相关规则：

- 代理只能在钱包余额足够时创建兑换码。
- 代理扣费公式：`quota * agent_discount% * count`。
- 代理创建兑换码不能设置过期时间。
- 代理不能兑换自己或其他代理生成的兑换码；代理只能兑换管理员/root 生成的兑换码。
- 删除未使用的代理兑换码时，退还原始 `agent_charge`，不是兑换码面值 quota。

搜索示例：

```bash
curl 'http://localhost:3000/api/redemption/search?keyword=agent001&p=1&page_size=20' \
  -H 'Cookie: <admin-session-cookie>' \
  -H 'New-Api-User: 1'
```

### 日志、使用量、数据 API

| 方法 | 路径 | Handler | 用途 | 权限 | 状态 |
| --- | --- | --- | --- | --- | --- |
| GET | `/api/log/` | `controller.GetAllLogs` | 管理员日志 | 管理员/root | 完成 |
| GET | `/api/log/stat` | `controller.GetLogsStat` | 管理员日志统计 | 管理员/root | 完成 |
| GET | `/api/log/self` | `controller.GetUserLogs` | 当前用户日志 | 用户 | 完成 |
| GET | `/api/log/self/stat` | `controller.GetLogsSelfStat` | 当前用户统计 | 用户 | 完成 |
| GET | `/api/log/search` | `controller.SearchAllLogs` | 已废弃搜索接口 | 管理员/root | 已废弃 |
| GET | `/api/log/self/search` | `controller.SearchUserLogs` | 已废弃用户搜索接口 | 用户 | 已废弃 |
| DELETE | `/api/log/` | `controller.DeleteHistoryLogs` | 旧版同步日志清理 | Root | 默认前端已不使用 |
| GET | `/api/log/channel_affinity_usage_cache` | `controller.GetChannelAffinityUsageCacheStats` | 渠道亲和缓存统计 | 管理员/root | 完成 |
| GET | `/api/log/token` | `controller.GetLogByKey` | Token 日志查询 | Token read-only | 完成 |
| GET | `/api/data/` | `controller.GetAllQuotaDates` | 额度日期数据 | 管理员/root | 完成 |
| GET | `/api/data/users` | `controller.GetQuotaDatesByUser` | 用户额度日期数据 | 管理员/root | 完成 |
| GET | `/api/data/self` | `controller.GetUserQuotaDates` | 当前用户额度日期数据 | 用户 | 完成 |
| GET | `/api/data/flow` | `controller.GetAllFlowQuotaDates` | 流量数据 | 管理员/root | 完成 |
| GET | `/api/data/flow/self` | `controller.GetUserFlowQuotaDates` | 当前用户流量数据 | 用户 | 完成 |

### 生图日志 API

以下接口均需要 `UserAuth()`。管理员/root 可查看全部日志，普通用户只能查看自己的记录；图片读取接口会再次校验日志所属用户。

| 方法 | 路径 | Handler | 用途 | 参数 | 响应 | 权限 | 状态 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| GET | `/api/image-generation-logs` | `controller.GetImageGenerationLogs` | 分页查询成功的同步生图记录 | `p`、`page_size`、`model`、`prompt`、`channel_id`、`start_timestamp`、`end_timestamp` | `PageInfo<ImageGenerationLog[]>`，每条包含受保护的 `image_urls`；无有效订阅权益返回 403 | 普通用户需有效订阅且只能看自己的最近 N 条；管理员/root 看全部 | 完成 |
| GET | `/api/image-generation-logs/` | `controller.GetImageGenerationLogs` | 兼容尾斜杠 | 同上 | 同上 | 同上 | 完成 |
| GET | `/api/image-generation-logs/:id/task` | `controller.GetImageGenerationLogTask` | 点击任务 ID 后读取当前状态和脱敏响应 JSON | path `id` | `{success,data: ImageGenerationTaskPayload}`；本地 base64 已替换为受保护图片 URL | 有效订阅所属用户或管理员/root | 完成 |
| GET | `/api/image-generation-logs/:id/images/:index` | `controller.GetImageGenerationLogImage` | 鉴权读取本地落盘图片 | path `id`、`index` | 图片二进制；不存在 404、无订阅/越权/超出最近 N 条范围均为 403 | 有效订阅所属用户或管理员/root | 完成 |

调用示例：

```bash
curl 'http://localhost:3000/api/image-generation-logs?p=1&page_size=20&model=gpt-image-2' \
  -H 'Cookie: <session-cookie>' \
  -H 'New-Api-User: 1'
```

### OpenAI 生图与异步轮询 API

MinIO 管理接口均需要 `AdminAuth()` 和 `operations.logs` 系统设置权限，已完成：

| 方法 | 路径 | Handler | 用途 | 请求/响应与错误处理 |
| --- | --- | --- | --- | --- |
| GET | `/api/performance/image-storage` | `controller.GetImageObjectStorageConfig` | 读取异步生图 MinIO 配置 | 无参数；返回 `{success,data: ImageObjectStorageConfig}` 及 `has_secret_key`，永不返回 Secret Key 原文；无权限返回 403。 |
| PUT | `/api/performance/image-storage` | `controller.SaveImageObjectStorageConfig` | 保存 MinIO 配置 | JSON 字段为 `enabled/endpoint/bucket/region/access_key/secret_key/use_ssl/use_path_style/object_prefix`；Secret Key 留空沿用现有值；启用时校验凭据、Endpoint 和前缀；非法配置返回 400。 |
| POST | `/api/performance/image-storage/test` | `controller.TestImageObjectStorageConfig` | 使用当前表单测试 Bucket | 请求结构同 PUT；只执行连接及 `BucketExists`，不会创建 Bucket 或修改权限；连接失败/Bucket 不存在返回 400。 |

```bash
curl -X PUT http://localhost:3000/api/performance/image-storage \
  -H 'Content-Type: application/json' \
  -H 'Cookie: <session-cookie>' \
  -H 'New-Api-User: 1' \
  -d '{"enabled":true,"endpoint":"https://media.example.com","bucket":"julong-media","region":"us-east-1","access_key":"ACCESS","secret_key":"SECRET","use_ssl":true,"use_path_style":true,"object_prefix":"generated/images"}'
```

#### `POST /v1/images/generations`

- Handler：`controller.RelayImageGeneration`；同步执行继续进入 `controller.Relay`，异步执行由 `runImageGenerationTask` 在本进程后台复用同一 Relay 链路。
- 权限：`Authorization: Bearer <API_KEY>`；执行 `TokenAuth`、模型请求限流和 `Distribute`，任务与 Token 所属用户绑定。
- 计费：异步任务仍使用原渠道选择、预扣、重试、结算和失败退款逻辑；不会创建第二套计费规则。
- 完成状态：`pending`、`processing`、`success`、`failed`。正确拼写是 `success`。

JSON 请求参数：

| 参数 | 类型 | 必填/默认 | 用途和约束 | 上游行为 |
| --- | --- | --- | --- | --- |
| `model` | string | 必填 | 图片模型名称，如 `gpt-image-2` | 参与渠道选择并按适配器映射 |
| `prompt` | string | 必填 | 生图提示词 | 转发 |
| `async` | boolean | 默认 `false` | 生图日志开启时，`true` 立即返回任务 ID 并在后台生成；关闭日志时该字段被忽略并保持同步响应 | Julong 控制字段，转发前删除 |
| `n` | integer | 默认 `1` | 图片数量；省略或 `0` 归一化为 `1`，最大 `128`；部分上游可能限制更小 | 转发并参与计费 |
| `size` | string | 模型默认 | 图片尺寸，如 `1024x1024`；DALL-E 尺寸受专用校验 | 转发 |
| `quality` | string | 模型默认 | 如 `standard`、`high`、`hd` | 转发 |
| `response_format` | string | 上游默认 | 常见值 `url` 或 `b64_json` | 转发；base64 日志会解码落盘 |
| `style` | JSON | 可选 | provider 风格参数 | 按适配器支持情况转发 |
| `user` | JSON | 可选 | 上游终端用户标识 | 按适配器支持情况转发 |
| `background` | JSON | 可选 | 背景配置 | 按模型/适配器支持情况转发 |
| `moderation` | JSON | 可选 | 内容审核模式 | 按模型/适配器支持情况转发 |
| `output_format` | JSON | 可选 | 输出格式，如 png/webp | 按模型/适配器支持情况转发 |
| `output_compression` | JSON | 可选 | 输出压缩质量 | 按模型/适配器支持情况转发 |
| `partial_images` | JSON | 可选 | 流式局部图片数量 | 按模型支持；异步模式不支持流式 |
| `stream` | boolean | 默认 `false` | 图片流式响应 | `async: true` 时必须为 `false`，否则 400 |
| `watermark` | boolean | 可选 | 水印开关 | 按适配器支持情况转发 |
| `watermark_enabled` | JSON | 可选 | 智谱 4V 等渠道水印参数 | provider 专用 |
| `user_id` | JSON | 可选 | provider 用户标识 | provider 专用 |
| `image` / `images` / `mask` | JSON | 可选 | 部分 provider 的参考图/遮罩扩展 | `/images/generations` 仅按适配器支持情况处理；标准编辑应使用 `/v1/images/edits` |
| `input_fidelity` | JSON | 可选 | 参考图保真参数 | 按模型/适配器支持情况转发 |
| `extra_fields` | JSON | 可选 | 已声明的 provider 扩展容器 | 按适配器实现处理 |

未声明字段会被 `ImageRequest.Extra` 接收；全局/渠道透传模式保留原请求字段，标准转换模式只保证声明字段和适配器明确支持的扩展字段。`async` 在所有模式下都不会发送给上游。

异步任务依赖“系统设置 → 运维 → 日志维护 → 记录生图日志”。该开关关闭时，带 `async: true` 的请求返回普通同步图片响应，不返回任务 ID，且不创建生图日志或落盘图片。

MinIO 启用后，异步任务会将 Base64 结果或可下载的上游 URL 写入共享私有 Bucket。对象键固定为 `<object_prefix>/YYYY/MM/DD/{sha256}.{ext}`；默认是 `generated/images/YYYY/MM/DD/{sha256}.{ext}`。任务成功响应的 `data[]` 除鉴权图片读取 `url` 外，还包含 `bucket`、`object_key`、`sha256`、`mime_type`、`size`，供 `canvas.julongkj.top` 登记同一对象。MinIO 上传失败时回退到原本的本地文件或上游 URL，避免已成功的生成任务因存储故障被改成失败。

同步请求示例（省略 `async`）：

```bash
curl https://api.julongkj.top/v1/images/generations \
  -H 'Content-Type: application/json' \
  -H 'Authorization: Bearer sk-xxx' \
  -d '{"model":"gpt-image-2","prompt":"一张极简产品海报","n":1,"size":"1024x1024"}'
```

异步提交示例：

```bash
curl https://api.julongkj.top/v1/images/generations \
  -H 'Content-Type: application/json' \
  -H 'Authorization: Bearer sk-xxx' \
  -d '{"model":"gpt-image-2","prompt":"一张极简产品海报","n":1,"async":true}'
```

异步提交成功返回 HTTP 202：

```json
{
  "task_id": "img_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
  "object": "image.generation.task",
  "status": "pending",
  "created_at": 1784422800,
  "poll_url": "/v1/images/generations/img_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
  "polling_interval_seconds": 15
}
```

#### `GET /v1/images/generations/:task_id`

- Handler：`controller.GetImageGenerationTask`。
- 权限：`TokenAuthReadOnly`；只能查询当前 API Key 所属用户的任务，越权任务统一返回 404。
- 请求参数：path `task_id`，无请求体。
- 响应：HTTP 200，包含 `task_id/object/status/progress/created_at/updated_at/request_id/model/image_count/data/error/response/polling_interval_seconds`。`response` 保留上游 JSON 元数据，但移除 `b64_json` 并动态改写图片 URL。
- 轮询建议：`pending`/`processing` 按响应中的 `polling_interval_seconds` 查询，默认 15 秒；`success`/`failed` 后停止。

```bash
curl https://api.julongkj.top/v1/images/generations/img_xxx \
  -H 'Authorization: Bearer sk-xxx'
```

成功响应示例：

```json
{
  "task_id": "img_xxx",
  "object": "image.generation.task",
  "status": "success",
  "progress": 100,
  "model": "gpt-image-2",
  "image_count": 1,
  "polling_interval_seconds": 15,
  "data": [{"url":"/v1/images/generations/img_xxx/images/0"}],
  "error": null,
  "response": {"created":1784422800,"data":[{"url":"/v1/images/generations/img_xxx/images/0"}]}
}
```

#### `GET /v1/images/generations/:task_id/images/:index`

- Handler：`controller.GetImageGenerationTaskImage`。
- 权限：默认通过 `middleware.ImageGenerationTaskImageAuth` 执行 `TokenAuthReadOnly`，且任务必须属于当前用户。Root/获授权管理员打开 `ImageGenerationLogImageAuthWhitelistEnabled` 后，精确匹配配置 IP 或浏览器 `Origin/Referer` 域名的请求可免 API Key；`0.0.0.0` 全部放行。免鉴权时任务 ID 作为不可猜测的读取凭证，不提供公开任务列表。
- 请求参数：path `task_id`、`index`；鉴权模式需要 `Authorization: Bearer <API_KEY>`，白名单模式无请求体和鉴权头要求。
- 响应：本地落盘图片二进制和实际 MIME；远程 URL 图片直接出现在任务 JSON 中，不经过该接口代理。
- 错误处理：图片索引非法返回 400；任务、图片或文件不存在返回 404；非白名单且缺少/无效 API Key 返回 401；用户或 IP 被禁用返回 403。

```bash
curl -L https://api.julongkj.top/v1/images/generations/img_xxx/images/0 \
  -H 'Authorization: Bearer sk-xxx' \
  -o generated-image.png
```

#### `GET /v1/images/generations/:task_id/images/:index/presign`

- Handler：`controller.GetImageGenerationTaskImagePresign`，状态为完成。
- 用途：为 MinIO 类型的异步生图结果生成临时直链，客户端无需接触 MinIO 凭据。
- 权限：`TokenAuthReadOnly`，并通过 `user_id + task_id` 强制校验任务属于当前 API Key 用户；不受图片读取白名单影响。
- 参数：path `task_id/index`；可选 query `expires_in`，默认 `3600` 秒，范围 `60-86400`。
- 响应：`{"url":"https://...","expires_in":3600,"expires_at":"RFC3339"}`。
- 错误处理：索引非法 400；任务/图片不存在或越权 404；图片不是 MinIO 引用 409；配置/签名失败 500；缺少或无效 API Key 401。

```bash
curl 'https://api.julongkj.top/v1/images/generations/img_xxx/images/0/presign?expires_in=900' \
  -H 'Authorization: Bearer sk-xxx'
```

错误处理：请求 JSON/参数非法、`async + stream` 返回 400；API Key 无效返回 401；用户/IP/Token 被禁用返回 403；模型无可用渠道通常返回 503；任务不存在或越权返回 404；异步执行中的上游/计费错误写入任务 `error.message` 并将状态更新为 `failed`。

### 客服联系方式 API

| 方法 | 路径 | 函数 | 用途 | 请求/响应 | 权限 | 状态 |
| --- | --- | --- | --- | --- | --- | --- |
| GET | `/api/support-contacts` | `controller.GetSupportContacts` | 获取概览页客服联系方式 | 响应 `SupportContact[]` | 登录用户 | 完成 |
| PUT | `/api/support-contacts` | `controller.UpdateSupportContacts` | 保存多条 QQ/微信/手机联系方式 | 请求 `{contacts:[{type,label,value}]}`；非法类型、空值、超长或超过 30 条时报错 | root 或拥有 `system_settings.content.support` 的管理员 | 完成 |

### 后台设置与运维 API

系统设置与运维接口（root 拥有全部权限；管理员按 `system_settings.<一级菜单>.<二级页面>` 权限访问）：

- `GET /api/option/`
- `PUT /api/option/`
- `POST /api/option/payment_compliance`
- `GET /api/option/channel_affinity_cache`
- `DELETE /api/option/channel_affinity_cache`
- `POST /api/option/rest_model_ratio`
- `POST /api/option/migrate_console_setting`
- `GET /api/option/waffo-pancake/catalog`
- `POST /api/option/waffo-pancake/pair`
- `POST /api/option/waffo-pancake/save`
- `POST /api/option/waffo-pancake/subscription-product`
- `GET /api/option/waffo-pancake/subscription-product-options`

其中 `GET/PUT /api/option/` 要求管理员传入 `section=<一级菜单>.<二级页面>`，后端会同时校验页面权限和配置键归属；`POST /api/option/migrate_console_setting` 仍仅限 root。

其他管理分组：

| 前缀 | 关键路由 | 权限 | 用途 |
| --- | --- | --- | --- |
| `/api/authz` | `GET /catalog` | 管理员/root | 权限目录 |
| `/api/custom-oauth-provider` | `POST /discovery`、`GET/POST/PUT/DELETE` | Root | 自定义 OAuth Provider |
| `/api/performance` | `GET /stats`、`DELETE /disk_cache`、`POST /reset_stats`、`POST /gc`、`GET/DELETE /logs`、`GET/PUT /image-storage`、`POST /image-storage/test` | Root 或对应系统设置权限 | 运行时性能、日志及生图对象存储维护 |
| `/api/ratio_sync` | `GET /channels`、`POST /fetch` | Root | 上游倍率同步 |
| `/api/group` | `GET /` | 管理员/root | 分组列表 |
| `/api/prefill_group` | `GET/POST/PUT/DELETE` | 管理员/root | 预填模型分组 |
| `/api/system-task` | `POST /log-cleanup`、`GET /list`、`GET /current`、`GET /:task_id` | Root | 异步维护任务 |
| `/api/system-info` | `GET /instances`、`DELETE /stale-instances`、`DELETE /instances/:node_name` | Root | 多实例状态 |

### 模型、渠道、部署 API

| 前缀 | 路由 | 权限 | 用途 | 状态 |
| --- | --- | --- | --- | --- |
| `/api/channel` | `GET/POST/PUT/DELETE`、`/search`、`/test`、状态/批量路由见 `router/api-router.go` 和 `router/channel-router.go` | 管理员/root 及 authz 权限 | 上游渠道管理 | 完成 |
| `/api/channel/:id/key` | `POST` | 管理员/root | 显示渠道 key | 完成 |
| `/api/models` | `GET /`、`/search`、`/:id`、`POST /`、`PUT /`、`DELETE /:id`、`/sync_upstream/*`、`/missing` | 管理员/root | 模型元数据/目录/部署元数据 | 完成 |
| `/api/vendors` | `GET /`、`/search`、`/:id`、`POST`、`PUT`、`DELETE` | 管理员/root | Vendor 元数据 | 完成 |
| `/api/deployments` | settings、test、CRUD、硬件/地区/副本辅助、日志/容器详情 | 管理员/root | IONet/模型部署管理 | 完成 |
| `/api/mj` | `GET /`、`GET /self` | 管理员或用户 | Midjourney 任务日志 | 完成 |
| `/api/task` | `GET /`、`GET /self` | 管理员或用户 | 通用异步任务日志 | 完成 |

### Relay 兼容 API

Relay 路由注册在 `router/relay-router.go`，使用 API key 鉴权 `middleware.TokenAuth()`、系统性能检查、模型限流和 `middleware.Distribute()` 渠道路由。

已支持：

- `GET /v1/models`
- `GET /v1/models/:model`
- `GET /v1beta/models`
- `GET /v1beta/openai/models`
- `GET /v1/realtime`
- `POST /v1/messages`
- `POST /v1/completions`
- `POST /v1/chat/completions`
- `POST /v1/responses`
- `POST /v1/responses/compact`
- `POST /v1/edits`
- `POST /v1/images/generations`
- `POST /v1/images/edits`
- `POST /v1/embeddings`
- `POST /v1/audio/transcriptions`
- `POST /v1/audio/translations`
- `POST /v1/audio/speech`
- `POST /v1/rerank`
- `POST /v1/engines/:model/embeddings`
- `POST /v1/models/*path`
- `POST /v1/moderations`
- `POST /pg/chat/completions`，用于登录后的 Dashboard Playground。

明确未实现：

- `/v1/images/variations`
- `/v1/files*`
- `/v1/fine-tunes*`
- `DELETE /v1/models/:model`

任务/视频路由：

- Suno-like：`/suno/submit/:action`、`/suno/fetch`、`/suno/fetch/:id`。
- Midjourney-compatible：`/mj/submit/*`、`/mj/task/*`、`/mj/image/:id`、`/mj/insight-face/swap`。
- 视频路由见 `router/video-router.go`：`/v1/video/generations`、`/v1/videos`、`/v1/videos/:task_id`、`/kling/v1/videos/*`、`/jimeng/*`。

## 数据模型与数据库结构

迁移在 `model/main.go` 中通过 `DB.AutoMigrate(...)` 执行。SQLite 下 `SubscriptionPlan` 有专门迁移处理。快速迁移 `migrateDBFast()` 会并发迁移模型表，新增模型时也必须同步更新。

### 核心模型

| 模型 | 文件 | 表用途 | 关键字段和约束 | 关系/说明 | 状态 |
| --- | --- | --- | --- | --- | --- |
| `User` | `model/user.go` | Dashboard 用户 | `id`、唯一索引 `username`、`password`、`display_name`、`role`、`status`、`email`、`quota`、`used_quota`、`request_count`、`group`、唯一 `aff_code`、`inviter_id`、`is_agent`、`agent_discount`、`agent_topup_link`、`stripe_customer`、`last_login_at`、IPv4/IPv6 `last_login_ip`、时间戳 | 列表/详情必须隐藏 password/access token。登录成功统一在 `setupLogin` 更新时间/IP；`shared_ip_user_count`、`last_login_ip_blocked`、`AdminPermissions` 均为 transient 字段。 | 活跃 |
| `BlockedIP` | `model/blocked_ip.go` | 全局 IP 黑名单 | 唯一 `ip varchar(45)`、索引 `user_id` 来源用户、索引 `operator_id` 操作者、`reason varchar(255)`、创建/更新时间 | IP 经 `net.ParseIP` 标准化；封禁状态使用 Redis 或 60 秒本地缓存。普通用户登录、注册、Dashboard 会话和 API Token 鉴权均拦截；管理员/root Dashboard 登录豁免。 | 活跃 |
| `Token` | `model/token.go` | API keys | `id`、`user_id`、`key`、`name`、额度字段、模型限制、group、allow IPs | Relay 鉴权和计费使用。 | 活跃 |
| `Channel` | `model/channel.go` | 上游 provider 渠道 | `id`、`type`、`key`、`base_url`、`models`、`group`、状态、priority/weight、计费/override 字段 | 由 distributor middleware 选择。敏感 key 需要脱敏。 | 活跃 |
| `Ability` | `model/ability.go` | 渠道/模型能力映射 | channel/model/group enabled 字段 | 用于模型可用性和路由。 | 活跃 |
| `Log` | `model/log.go` | 使用日志和审计日志 | `id`、`user_id`、`created_at`、`type`、`content`、`username`、`token_name`、`model_name`、`quota`、`prompt_tokens`、`completion_tokens`、`channel_id`、`ip`、`request_id`、`upstream_request_id`、`other` | 可位于独立日志库/ClickHouse。`GetUserLoginIPStats` 聚合登录 IP、次数和最后时间；`SumUserUsedToken` 提供总 token 消耗。 | 活跃 |
| `ImageGenerationLog` / `ImageGenerationImage` | `model/image_generation_log.go` | 同步生图日志与本地异步任务 | 日志含 `id`；索引 `task_id/status/user_id/username/token_id/channel_id/model_name/request_id/created_at/updated_at`；`prompt/size/quality/image_count/quota/use_time`；内部 `images/response` JSON。`ImageGenerationImage` JSON 字段为 `type/value/bucket/mime_type/sha256/size/revised_prompt`，`type` 支持 `local/remote/minio`。 | 任务属于一个 User/Token；MinIO 元数据写入现有 `images` 文本 JSON，不新增数据库列或表。旧 local/remote JSON 继续兼容；AutoMigrate 无额外迁移。 | 活跃 |
| `Redemption` | `model/redemption.go` | 兑换码 | 唯一 `key`、`user_id` 生成者、`status`、`name`、`quota`、`created_time`、`redeemed_time`、`used_user_id`、`expired_time`、`agent_charge`、软删除 | 生成者显示字段是 transient。搜索支持 code、生成者、ID、名称。代理退款使用 `agent_charge`。 | 活跃 |
| `TopUp` | `model/topup.go` | 在线支付/充值订单 | trade no、user、amount/money、provider、status | 支付回调结算。 | 活跃 |
| `Checkin` / `CheckinRecord` | `model/checkin.go` | 每日签到奖励记录 | user/date/quota/time 字段 | 设置位于 `checkin_setting.*`。 | 活跃 |
| `Model` | `model/model_meta.go` | 模型目录元数据 | 模型名、vendor、展示元数据、定价/可见性 | 用于模型广场和后台模型元数据。 | 活跃 |
| `Vendor` | `model/vendor_meta.go` | Vendor 元数据 | id/name/display/config 字段 | 与 Model 元数据关联。 | 活跃 |
| `Pricing` / `PricingVendor` | `model/pricing.go` | 定价页面数据 | 模型定价和 vendor 分组字段 | 用于公开定价页。 | 活跃 |
| `PrefillGroup` | `model/prefill_group.go` | 模型预填分组 | group name、models/config | 后台模型管理。 | 活跃 |
| `Task` | `model/task.go` | 异步模型/媒体任务 | platform、task id、action、status、quota/billing private data | 用于视频/音乐/图片任务 relay。 | 活跃 |
| `Midjourney` | `model/midjourney.go` | MJProxy 风格任务 | mj id/action/status/prompt/result 字段 | 兼容/旧版图片任务跟踪。 | 活跃 |
| `SubscriptionPlan` | `model/subscription.go` | 订阅计划 | id/title/price/quota/reset/status，以及 `allow_image_generation_logs`、`image_generation_log_limit`（0 为不限条数） | SQLite 有自定义迁移；生图日志权益默认关闭。 | 活跃 |
| `SubscriptionOrder` | `model/subscription.go` | 订阅支付订单 | order id/user/plan/payment status 字段 | 支付回调使用。 | 活跃 |
| `UserSubscription` | `model/subscription.go` | 用户有效订阅 | user/plan/quota/period/status，以及生图日志权限和条数快照字段 | 创建订阅时从套餐复制权益；多个有效订阅任一为 0 则无限，否则取最大条数。 | 活跃 |
| `SubscriptionPreConsumeRecord` | `model/subscription.go` | 订阅预扣记录 | subscription/request/pre/post quota 字段 | 请求结算使用。 | 活跃 |
| `Option` | `model/option.go` | 运行时设置 | `key`、`value` | Root 或获授权管理员通过设置 API 修改；`console_setting.announcements` 保存公告 JSON 配置。 | 活跃 |
| `Setup` | `model/setup.go` | 安装/初始化状态 | setup timestamp/status | `/api/setup`。 | 活跃 |
| `PasskeyCredential` | `model/passkey.go` | WebAuthn 凭据 | user/credential 字段 | Passkey 登录。 | 活跃 |
| `TwoFA` / `TwoFABackupCode` | `model/twofa.go` | 2FA 密钥和备份码 | user secret/status/codes | 2FA 登录和管理员重置。 | 活跃 |
| `CustomOAuthProvider` | `model/custom_oauth_provider.go` | Root 可配置 OAuth provider | provider id/name/client ids/discovery/policy 字段 | 自定义 OAuth 管理。 | 活跃 |
| `UserOAuthBinding` | `model/user_oauth_binding.go` | OAuth 账号绑定 | user/provider/external id | 用户/管理员解绑。 | 活跃 |
| `PerfMetric` | `model/perf_metric.go` | 性能采样 | route/model/provider/time/error/cache 字段 | 公开性能指标/定价页。 | 活跃 |
| `SystemInstance` | `model/system_instance.go` | 运行节点 | node name/status/time 字段 | 多实例状态。 | 活跃 |
| `SystemTask` / `SystemTaskLock` | `model/system_task.go` | 后台维护任务 | task id/type/status/payload/state/lock | Root 运维操作。 | 活跃 |
| `CasbinRule` / `AuthzRole` | `model/casbin_rule.go`、`model/authz_role.go` | 细粒度管理员权限 | casbin p/v 字段和角色分配 | Admin authz catalog。 | 活跃 |
| `ErrorReport` | `model/error_report.go` | 500 页面反馈 | `id`、`created_at`、索引 `user_id`、`username`、`title`、`message`、`page_url`、`error_status`、`user_agent`、`stack`、`ip` | Julong 二开。已加入两种迁移流程。 | 活跃 |
| `QuotaData` / `FlowQuotaData` | `model/usedata.go`、`model/usedata_flow.go` | Dashboard 聚合用量 | date/user/quota/flow 字段 | 数据看板。 | 活跃 |

`setting/console_setting/config.go:ConsoleSetting` 是存储于 `Option` 的运行时 JSON 配置结构，不单独建表。`custom_endpoints` 保存最多 20 条 `{id,name,url,description}`，经 URL、长度和危险内容校验后通过公开状态接口下发。公告相关字段为 `announcements`（JSON 数组字符串）和 `announcements_enabled`（公告总开关）。单条公告包含 `id/title/content/status/notificationMode/startTime/endTime/audienceMode/conditionGroups`；状态为 `draft/active/archived`，通知方式为 `silent/popup`，条件组之间为 OR、组内条件为 AND，当前支持订阅套餐包含/排除和余额比较。旧公告读取时自动归一化为展示中、静默、所有用户，无需数据库迁移。

余额条件的阈值使用前端展示额度单位；匹配时后端以 `user.quota / QuotaPerUnit` 换算。套餐条件只匹配状态为 active 且未过期的 `UserSubscription.plan_id`。开始/结束时间使用 RFC3339，空开始时间表示立即生效，空结束时间表示永久有效。

### 迁移注意事项

- 新增持久化模型时，必须同时加入 `model/main.go` 的 `migrateDB()` 和 `migrateDBFast()`。
- 日志表结构变化可能还需要更新 `migrateLOGDB()`。
- PostgreSQL 保留字字段（如 `key`）要像 `model/redemption.go` 一样做数据库类型判断并加引号。
- SQLite 字段变更必须用已有本地 `one-api.db` 测试迁移。
- 仅用于 JSON 响应、不入库的字段要加 `gorm:"-:all"`。
- `User.last_login_ip` 由现有 `User` AutoMigrate 在服务启动时自动补列，无需注册新的迁移模型；长度 45，可保存 IPv4 和 IPv6。
- `BlockedIP` 已加入 `migrateDB()` 和 `migrateDBFast()`；部署新版本启动后自动创建 `blocked_ips` 表，不改动现有用户和日志数据。

## 前端路由

路由位于 `web/default/src/routes`。

| 路由 | 文件 | 组件/功能 | 权限 | 状态 |
| --- | --- | --- | --- | --- |
| `/` | `routes/index.tsx` | 首页 | 公开 | 完成 |
| `/sign-in`、`/sign-up`、`/forgot-password`、`/reset`、`/otp` | `routes/(auth)/*` | 认证功能 | 公开 | 完成 |
| `/setup` | `routes/setup/index.tsx` | 初始化向导 | 公开/安装 | 完成 |
| `/pricing`、`/pricing/:modelId` | `routes/pricing/*` | 定价/模型广场 | 公开或导航权限 | 完成 |
| `/rankings` | `routes/rankings/index.tsx` | 排行榜 | 公开或导航权限 | 完成 |
| `/about`、`/privacy-policy`、`/user-agreement` | legal/about routes | 法务/内容 | 公开 | 完成 |
| `/error-report` | `routes/error-report.tsx` | 提交错误反馈 | 公开 | 完成 |
| `/dashboard/*` | `_authenticated/dashboard/*` | 看板统计 | 用户 | 完成 |
| `/keys` | `_authenticated/keys/index.tsx` | API Keys | 用户 | 完成 |
| `/usage-logs/*` | `_authenticated/usage-logs/*` | 使用日志/任务日志/绘图日志 | 用户/管理员视图不同 | 完成 |
| `/wallet` | `_authenticated/wallet/index.tsx` | 钱包、充值、兑换码 | 用户 | 完成 |
| `/profile` | `_authenticated/profile/index.tsx` | 个人资料/安全 | 用户 | 完成 |
| `/agent-users` | `_authenticated/agent-users/index.tsx` | 代理所属用户 | 代理 | 完成 |
| `/redemption-codes` | `_authenticated/redemption-codes/index.tsx` | 兑换码管理 | 用户；管理员全部，代理限自己 | 完成 |
| `/channels` | `_authenticated/channels/index.tsx` | 渠道管理 | 管理员/root | 完成 |
| `/models/*` | `_authenticated/models/*` | 模型元数据/部署 | 管理员/root | 完成 |
| `/users` | `_authenticated/users/index.tsx` | 用户管理 | 管理员/root | 完成 |
| `/subscriptions` | `_authenticated/subscriptions/index.tsx` | 订阅管理 | 管理员/root | 完成 |
| `/error-reports` | `_authenticated/error-reports/index.tsx` | 错误反馈列表 | 管理员/root | 完成 |
| `/system-info` | `_authenticated/system-info/index.tsx` | 系统节点/任务 | Root | 完成 |
| `/system-settings/*` | `_authenticated/system-settings/*` | 系统设置 | 管理员/root，视页面而定 | 完成 |
| `/401`、`/403`、`/404`、`/500`、`/503` | `routes/(errors)/*` | 错误页 | 公开 | 完成 |

## 前端组件清单

### 共享组件分组

以下路径均位于 `web/default/src/components`。

| 分组 | 文件 | 用途 | 依赖 | 状态 |
| --- | --- | --- | --- | --- |
| UI primitives | `ui/accordion.tsx`、`alert-dialog.tsx`、`alert.tsx`、`aspect-ratio.tsx`、`avatar.tsx`、`badge.tsx`、`breadcrumb.tsx`、`button.tsx`、`button-group.tsx`、`calendar.tsx`、`card.tsx`、`carousel.tsx`、`chart.tsx`、`checkbox.tsx`、`collapsible.tsx`、`combobox.tsx`、`combobox-input.tsx`、`command.tsx`、`context-menu.tsx`、`dialog.tsx`、`drawer.tsx`、`empty.tsx`、`field.tsx`、`form.tsx`、`hover-card.tsx`、`input.tsx`、`input-group.tsx`、`input-otp.tsx`、`item.tsx`、`kbd.tsx`、`label.tsx`、`markdown.tsx`、`menubar.tsx`、`native-select.tsx`、`navigation-menu.tsx`、`pagination.tsx`、`popover.tsx`、`progress.tsx`、`radio-group.tsx`、`resizable.tsx`、`scroll-area.tsx`、`select.tsx`、`separator.tsx`、`sheet.tsx`、`sidebar.tsx`、`skeleton.tsx`、`slider.tsx`、`sonner.tsx`、`spinner.tsx`、`switch.tsx`、`table.tsx`、`tabs.tsx`、`textarea.tsx`、`titled-card.tsx`、`toggle.tsx`、`toggle-group.tsx`、`tooltip.tsx` | 可复用设计系统 primitives。关键参数通常透传到底层 Base UI/Radix-like primitive 并支持 `className`；Button 支持 variants/sizes/render/nativeButton。 | Base UI、Hugeicons/lucide、Tailwind、`cn` | 完成 |
| Data table | `data-table/core/*`、`data-table/layout/*`、`data-table/toolbar/*`、`data-table/hooks/*`、`data-table/static/*` | 响应式表格系统：桌面表格、移动卡片、分页、工具栏、过滤器、视图模式、批量操作。 | TanStack Table、media query hooks、UI primitives | 完成 |
| Layout | `layout/components/*`、`layout/config/*`、`layout/lib/*`、`layout/types.ts` | 登录后/公开布局、侧边栏、顶部导航、SectionPageLayout、系统品牌。 | TanStack Router、sidebar config hooks、auth store | 完成 |
| AI elements | `ai-elements/*` | Playground/chat 响应渲染、prompt input、artifact、reasoning、tool、sources、code block。 | React、markdown/shiki、AI SDK 风格组件 | 完成 |
| Utility widgets | `announcement-popup.tsx`、`copy-button.tsx`、`confirm-dialog.tsx`、`date-picker.tsx`、`datetime-picker.tsx`、`empty-state.tsx`、`error-state.tsx`、`group-badge.tsx`、`json-editor.tsx`、`json-code-editor.tsx`、`language-switcher.tsx`、`long-text.tsx`、`masked-value-display.tsx`、`model-group-selector.tsx`、`multi-select.tsx`、`password-input.tsx`、`profile-dropdown.tsx`、`provider-badge.tsx`、`status-badge.tsx`、`table-id.tsx`、`tag-input.tsx`、`theme-switch.tsx`、`turnstile.tsx` | 跨功能控件和显示组件；`AnnouncementPopup` 读取 `/api/announcements`，仅弹出 `notificationMode=popup` 的公告，并按登录用户和公告内容签名在 localStorage 记录关闭状态。 | UI primitives、React Query、auth store、i18n、本地格式化工具 | 完成 |

### Feature 模块

以下路径均位于 `web/default/src/features`。

| Feature | 关键文件/组件 | 用途 | API 依赖 | 状态 |
| --- | --- | --- | --- | --- |
| `auth` | `sign-in`、`sign-up`、`forgot-password`、`otp`、`passkey`、`secure-verification`、`components/oauth-providers.tsx` | 登录、注册、找回密码、OAuth、Passkey、2FA | `/api/user/*`、`/api/oauth/*`、`/api/verify` | 完成 |
| `home` | `index.tsx`、hero/gateway/stat 组件 | 公开首页内容 | `/api/home_page_content` | 完成 |
| `dashboard` | `index.tsx`、`section-registry.tsx`、stats/charts libs | 用户/管理员看板统计 | `/api/data*`、`/api/dashboard*`、`/api/status` | 完成 |
| `announcements` | `api.ts`、`types.ts` | 公告类型和当前用户公告查询；供顶部通知中心、自动弹窗和后台公告编辑器共享 | `/api/announcements` | 完成 |
| `custom-endpoints` | `types.ts`、`system-settings/content/custom-endpoints-section.tsx`、`keys/components/custom-endpoints.tsx` | 自定义端点共享类型、后台编辑器、API 密钥页复制条和悬停介绍 | `/api/option`、`/api/status` | 完成 |
| `channels` | `channels-table.tsx`、`channels-columns.tsx`、dialogs/drawers、`api.ts` | 上游渠道 CRUD/测试/配置 | `/api/channel*` | 完成 |
| `keys` | `api-keys-table.tsx`、`api-keys-columns.tsx`、mutate/delete dialogs | 用户 API key 管理 | `/api/token*` | 完成 |
| `usage-logs` | `usage-logs-table.tsx`、普通/绘图/生图/任务 columns、`image-generation-task-dialog.tsx`、图片预览和筛选组件 | 普通消费日志、Midjourney 绘图日志、同步/异步生图日志、媒体任务日志；未完成生图按 root 配置频率刷新（默认 15 秒），任务 ID 弹窗由页面级状态持有并按需请求 JSON，不受表格轮询重建影响 | `/api/log*`、`/api/mj`、`/api/image-generation-logs*`、`/api/task` | 完成 |
| `wallet` | recharge cards、subscription cards、affiliate rewards、redemption hook | 钱包充值、兑换码、订阅 | `/api/user/topup*`、`/api/subscription*`、支付 API | 完成 |
| `redemption-codes` | `redemptions-table.tsx`、`redemptions-columns.tsx`、mutate/delete dialogs | 管理员/代理兑换码管理 | `/api/redemption*`、`/api/user/agent/topup-link` | 完成 |
| `users` | `users-table.tsx`、`users-columns.tsx`、`users-mutate-drawer.tsx`、`agent-detail-dialog.tsx`、`user-detail-dialog.tsx` | 后台用户管理、代理详情、用户详情；详情弹窗包含头像身份摘要、关键指标带、订阅摘要、紧凑信息网格、响应式数据表和加载骨架 | `/api/user*`、`/api/log`、`/api/subscription/admin/*` | 完成 |
| `models` | metadata/deployment tables and drawers | 模型元数据和部署管理 | `/api/models*`、`/api/vendors*`、`/api/deployments*` | 完成 |
| `subscriptions` | subscription table/drawers | 后台订阅计划/用户绑定 | `/api/subscription/admin*` | 完成 |
| `system-settings` | `auth`、`billing`、`content`、`models`、`request-limits`、`maintenance`、`integrations`、`general` 下的 section registries；`maintenance/log-settings-section.tsx:LogSettingsSection` | 管理员/root 运行时设置；`LogSettingsSection` 负责消费日志、生图日志、保留天数、轮询频率和图片读取白名单，关键 props 为对应 `default*` 配置，依赖 React Hook Form/Zod、`useUpdateOption`、Switch/Textarea/Alert | `/api/option*` 及特定后台 API | 完成/进行中；图片读取白名单已完成 |
| `system-info` | system instances/tasks panels | Root 系统运维 | `/api/system-info*`、`/api/system-task*` | 完成 |
| `error-reports` | `submit-error-report.tsx`、`index.tsx`、`api.ts` | 提交/查看 500 页面反馈 | `/api/error-reports*` | 完成 |
| `errors` | `general-error.tsx`、forbidden/not-found/maintenance/unauthorized | 错误页 | `/error-report` 路由用于反馈 | 完成 |
| `pricing` | pricing tables/model detail | 公开模型定价 | `/api/pricing` | 完成 |
| `rankings` | hero/model leaderboard/provider sections | 公开排行榜 | `/api/rankings` | 完成 |
| `playground` | chat playground UI | 浏览器内请求测试 | `/pg/chat/completions` | 完成 |
| `setup` | setup wizard steps | 初始化安装 | `/api/setup` | 完成 |
| `profile` | profile/security settings | 用户资料 | `/api/user/self`、passkey/2FA APIs | 完成 |
| `legal`、`about` | legal/about documents | 静态/服务端内容页 | `/api/about`、`/api/privacy-policy`、`/api/user-agreement` | 完成 |

## Julong 二开功能

### 品牌

- 产品名显示为 `矩龙-API`。
- Docker 镜像/tag 目标为 `qq1371446705/julong-api:latest`。
- Compose project/container 使用 `julong-api`。
- 已知技术债：Go module import path 仍为上游 `github.com/QuantumNous/new-api`；重命名风险较高，暂不计划。

### 代理系统

相关文件：

- 后端：`model/user.go`、`controller/user.go`、`controller/redemption.go`、`model/redemption.go`。
- 前端：`web/default/src/features/users/*`、`web/default/src/features/redemption-codes/*`、`web/default/src/features/wallet/*`。

行为：

- 管理员/root 可将用户设置为代理，并配置 0-100 的 `agent_discount`。
- 代理可生成兑换码。钱包扣费公式：`quota * agent_discount% * count`。
- 代理创建兑换码会检查钱包余额，不使用订阅额度。
- 代理生成的兑换码记录 `agent_charge`。
- 管理员/root 删除未使用的代理兑换码时，退还原始 `agent_charge`。
- 代理使用日志包含退款记录，代理和管理员/root 都能查看。
- 代理可以只读查看自己的所属用户。
- 代理可以设置 `agent_topup_link`；通过该代理邀请的用户在钱包兑换码区域看到代理自己的购买链接。

### 错误反馈流程

相关文件：

- 后端：`model/error_report.go`、`controller/error_report.go`、`router/api-router.go`、`model/main.go` 中的迁移。
- 前端：`web/default/src/features/errors/general-error.tsx`、`web/default/src/features/error-reports/*`、`web/default/src/routes/error-report.tsx`、`web/default/src/routes/_authenticated/error-reports/index.tsx`。

行为：

- 500 页面按钮跳转到 `/error-report`。
- 公开用户可以提交错误详情；已登录时通过 `TryUserAuth` 记录 user ID/username。
- 管理员/root 在 `/error-reports` 查看列表和详情。

### 用户详情弹窗

相关文件：

- 前端：`web/default/src/features/users/components/user-detail-dialog.tsx`、`users-columns.tsx`、`users/index.tsx`。
- 后端：`controller/user.go:AdminGetUserUsageSummary/AdminGetUserLoginIPs/AdminUpdateUserLoginIPs`、`model/log.go:GetUserLoginIPStats/SumUserUsedToken`、`model/blocked_ip.go`。

行为：

- 后台用户表点击用户名打开详情弹窗。
- 显示基本信息、最近登录时间和 IP、动态计算的未登录天数、格式化额度、请求数、总 token 消耗及最近 20 条消费日志；无成功登录记录时显示“从未登录”。
- 总 token 消耗按 `user_id` 从日志表求和。
- 代理详情和用户详情采用统一的紧凑弹窗布局：身份摘要置顶，关键额度/用量指标独立展示，基本信息使用单一网格面板。
- 弹窗限制在可视区域内滚动；移动端详情单列显示，日志、兑换码和所属用户表格支持横向滚动。
- 用户详情基本信息区显示有效订阅摘要；前端并行读取用户订阅与套餐列表，展示套餐名称、状态、剩余额度、到期时间和有效订阅数量。
- “登录 IP”页签聚合每个历史 IP 的登录次数和最后使用时间，可多选封禁/解封；同一最近登录 IP 被多个未删除用户使用时显示共享人数。
- `LoginIPPanel` 依赖 `getUserLoginIPs/updateUserLoginIPs` 和 React Query，关键参数为 `userId/onUpdated`；`LoginIPRow` 负责移动端/桌面 IP、次数、时间、共享和封禁状态展示，均已完成。
- 禁用用户会在同一个主库事务中禁用账号并封禁该用户全部登录审计 IP 及最近 IP。重新启用账号不会自动解封 IP，必须在登录 IP 页签手动选择解封。
- 用户列表执行禁用前显示破坏性确认弹窗，明确提示 IP 联动封禁及重新启用不会自动解封。
- 黑名单阻止普通用户注册、登录、已有 Dashboard 会话和 API Token 请求；管理员/root 的 Dashboard 登录豁免，避免共享办公出口 IP 导致后台锁死。

### 兑换码搜索增强

相关文件：

- 后端：`model/redemption.go:SearchRedemptionsByUser`。
- 前端：`web/default/src/features/redemption-codes/components/redemptions-table.tsx`。

行为：

- 搜索关键词匹配 ID、名称、兑换码 key/code、生成者用户名、生成者显示名。
- 状态过滤保持兼容。
- PostgreSQL 下 `key` 保留字段引用已处理。

### 签到额度预览

相关文件：

- `web/default/src/features/system-settings/general/checkin-settings-section.tsx`。

行为：

- 最小和最大签到额度说明显示格式化额度预览，使用 `formatQuota`，与额度设置页面一致。

### 生图日志

相关文件：

- 后端：`model/image_generation_log.go`、`service/image_generation_log.go`、`controller/image_generation_log.go`、`controller/image_generation_task.go`、`relay/image_handler.go` 及各图片渠道响应适配器。
- 前端：`web/default/src/features/usage-logs/components/columns/image-generation-logs-columns.tsx`、`image-generation-task-dialog.tsx`、`image-generation-preview-dialog.tsx`、日志 section/filter/API/types；root 日志维护设置。

行为：

- Root 在“系统设置 → 运营设置 → 日志维护”开启“记录生图日志”，默认关闭。
- 成功的 `/v1/images/generations` 请求，以及聊天/Responses 中成功返回 `image_generation_call.result` 的生图调用会写入记录；`/v1/images/edits` 暂不写入生图日志。
- 记录用户、Token、渠道、模型、提示词、尺寸、品质、图片数、费用、请求 ID 和耗时。
- base64 结果先解码为原始图片文件，单张上限 25MB；数据库只保存图片引用，避免 base64 约 33% 的编码膨胀。
- MinIO 关闭时，上游 URL 结果仍保存 URL 引用，Base64 结果保存在本地目录；MinIO 开启时，服务端受限下载上游图片或解码 Base64 后按 SHA-256 写入共享私有 Bucket，任务/日志图片接口再从对象存储读取，客户端不接触 MinIO 凭据。
- MinIO 对象不会随聚合平台生图日志过期清理直接删除，因为同一个 `object_key` 可能已被画布媒体资产引用；后续统一由画布资产引用计数和 Bucket 生命周期策略处理。
- 本地异步任务图片读取可配置免鉴权白名单：IP 精确匹配客户端 IP，域名精确匹配浏览器 `Origin/Referer` 主机名，`0.0.0.0` 放行所有图片读取；任务查询和图片生成始终保留 API Key 鉴权。域名请求头可由非浏览器客户端伪造，因此敏感部署优先使用 IP 白名单，并确保反向代理正确覆盖真实客户端 IP 头。
- 普通用户必须拥有启用生图日志权限的有效订阅，且只能查看自己的图片；管理员/root 可查看全部。
- 套餐可设置允许查看的最近日志条数，`0` 为全部；多个有效订阅任一为 `0` 时无限制，否则取最大值。列表查询和图片读取接口都会校验该范围，不能通过旧图片 URL 绕过。
- 生图日志预览弹窗支持逐张下载图片，并以 JSON 页签展示/下载脱敏后的日志元数据；本地文件路径和数据库内部图片引用不暴露给前端。
- 默认保留 30 天；写入新记录时每小时至多触发一次过期日志和本地图片清理。
- `/v1/images/generations` 支持可选 `async: true`。仅在记录生图日志开启时创建任务记录；关闭时退回同步且不存图。API Key 可轮询自己的任务，后台日志可点击任务 ID 查看实时状态和脱敏 JSON。
- 异步执行最多同时运行 16 个本地任务；任务复用同步 Relay，因此计费和退款行为一致。列表只在存在未完成任务时按 `ImageGenerationLogPollingIntervalSeconds` 自动刷新，默认 15 秒。

## 已知问题

| 问题 | 影响 | 当前处理方式 | 建议修复 |
| --- | --- | --- | --- |
| `docker-compose.dev.yml` 仍使用 `new-api-dev` 名称和数据库名 `new-api` | 开发 Compose 品牌名不一致 | 使用生产 Compose 或手动调整 dev compose | 将 dev compose 的 service/image/db 名全部改为 Julong 命名。 |
| Go module path 仍为 `github.com/QuantumNous/new-api` | 内部 import 仍显示上游名 | 为稳定性暂时保留 | 仅在准备好更新所有 imports、CI、Docker、上游合并策略时再重命名。 |
| 部分路由/API 文档按路由组汇总 | Relay 表面很大，完全展开会很长 | 用 `rg ".(GET|POST|PUT|PATCH|DELETE)(" router` 查看精确源码 | 如需机器可读规范，增加生成式 API 附录。 |
| 语言包容易被误手动编辑 | 可能漏翻译或破坏排序 | 必须使用临时脚本 + `bun run i18n:sync` | 增加 lint/precommit 校验语言包一致性。 |
| 生产 Compose 包含示例密码 | 若直接公开部署存在安全风险 | 部署前手动修改密码 | 移到 `.env` 或 Docker secrets。 |
| 错误反馈提交接口公开 | 可能被刷反馈 | 当前有全局限流和 body limit | 如被滥用，增加专用限流或验证码。 |
| 代理规则需要更多回归测试 | 计费/退款 bug 风险高 | 当前依赖手动 QA 和已有测试 | 增加 agent 创建/删除/退款/兑换限制的 model/controller 测试。 |
| 异步生图执行器是进程内任务 | 容器在上游请求执行期间重启时，该请求无法自动恢复，日志可能停留在 `pending/processing` | 正常完成/失败会更新状态；图片和任务记录已持久化 | 后续改为 Redis/数据库持久化队列，增加租约、超时失败回收和多节点 worker。 |
| 未启用 Redis 的多实例部署中，IP 黑名单负缓存最多延迟 60 秒同步 | 另一实例可能短暂继续放行刚封禁的 IP | 单实例立即失效；Redis 部署使用共享缓存并立即失效 | 生产多实例必须启用 Redis，或增加数据库通知机制。 |

## 技术债

- 将 `router/api-router.go` 按领域拆分成多个路由注册文件。
- 从 Gin 路由生成 OpenAPI 或 route manifest。
- 统一 API 错误 HTTP 状态；当前许多业务错误返回 HTTP 200。
- 合并或删除已废弃的 admin/user log search 接口。
- 为所有 controller 请求/响应增加明确 DTO，减少直接暴露 model JSON。
- 对高风险 schema 变更引入迁移文件，而不是只依赖 AutoMigrate。
- 对关键后台页面增加前端视觉回归检查。
- 继续优化所有后台表格移动端布局，而不只限兑换码。

## 进度

### 已完成

- Julong 生产 Compose Docker 命名和镜像配置。
- README 中记录本地 Docker 镜像维护流程。
- 代理折扣、代理生成兑换码、代理用户、代理充值链接流程。
- 代理兑换码删除时按原始 `agent_charge` 退款。
- 代理/管理员可见的退款使用日志。
- 兑换码列表移动端操作优化。
- 从 500 页面提交/后台查看错误反馈。
- 后台用户列表点击用户名查看用户详情。
- 管理员/root 可查看、批量封禁/解封用户历史登录 IP；禁用用户自动封禁其已有 IP，用户列表标记共享和已封 IP。
- 兑换码支持按生成者、兑换码、名称、ID 搜索。
- 签到奖励额度预览。
- 可由 root 开关控制、支持图片预览和自动保留清理的生图日志。
- 生图 `async: true` 提交、任务 ID 轮询、状态/错误记录、受保护图片读取和后台 JSON 详情；轮询频率可配置，日志关闭时异步和图片存储同步关闭。
- 订阅套餐可授予普通用户生图日志查看权限，并限制可见的最近记录数量。
- 钱包页优化统计区移动端布局、宽屏双栏比例和套餐卡片信息层级；单个套餐不再保留空白列，并在可购套餐和当前订阅中展示生图日志权益。
- 新增 UI 文案的多语言同步。

### 进行中

- 本 `DEVELOPMENT.md` 项目文档维护。
- 代理和计费逻辑持续 QA。
- 服务器部署加固和域名/反代配置。

### 待开发

- 覆盖所有自定义代理计费/退款路径的自动化测试。
- 错误反馈数量/未读状态后台看板组件。
- 错误反馈状态流：open/resolved/ignored。
- 更细的兑换码生成者和兑换码字段搜索/过滤 UI。
- Julong 生产 `.env` 模板。
- CI 流程：Go tests、前端 typecheck、build、Docker image push。

## 变更日志

| 日期 | 变更 | 更新文件/API/模型 | 验证 |
| --- | --- | --- | --- |
| 2026-07-19 | 日志维护的“记录生图日志”增加可保存、测试连接的 MinIO 配置；异步生图完成后将 Base64 或可下载 URL 按 SHA-256 写入共享私有 Bucket，任务响应返回统一对象元数据，读取接口兼容本地/远程/MinIO 三种历史引用，并提供所有者专用预签名接口。Secret Key 不回显，留空沿用；存储失败保留原结果回退；远程图片下载使用 SSRF 防护客户端。 | `service/image_object_storage.go`、`controller/image_object_storage.go`、`GET/PUT /api/performance/image-storage`、`POST /api/performance/image-storage/test`、`GET /v1/images/generations/:task_id/images/:index/presign`、`ImageGenerationImage`、`service/image_generation_log.go`、`LogSettingsSection`、locale files、MinIO Go SDK | `go test ./...`、`bun run typecheck`、目标 `oxfmt/oxlint`、`bun run build`、`bun run i18n:sync`、`git diff --check` |
| 2026-07-19 | 日志维护新增生图图片读取免鉴权白名单开关和多域名/IP 配置；精确匹配客户端 IP 或浏览器 Origin/Referer，`0.0.0.0` 全放行。仅图片二进制接口可绕过鉴权，任务 JSON 查询和生图提交保持 API Key/计费保护。 | `ImageGenerationLogImageAuthWhitelistEnabled`、`ImageGenerationLogImageAuthWhitelist`、`common/image_generation_image_auth_whitelist.go`、`middleware.ImageGenerationTaskImageAuth`、`GET /v1/images/generations/:task_id/images/:index`、`LogSettingsSection`、locale files、`生图轮询接口说明.md` | `go test ./...`、`bun run typecheck`、目标 `oxlint`/`oxfmt`、`bun run build`、`bun run i18n:sync`、`git diff --check` |
| 2026-07-19 | 新增独立的生图轮询接口交付文档，覆盖启用条件、完整请求参数、任务提交/查询响应、自动轮询、图片下载、错误处理、存储与进程限制。 | `生图轮询接口说明.md` | 文档内容与当前 API、配置和 `DEVELOPMENT.md` 逐项核对，`git diff --check` |
| 2026-07-19 | 将未完成生图任务轮询从固定 2 秒改为默认 15 秒，并允许在日志维护中配置 5-3600 秒；任务响应公开建议频率。记录生图日志关闭时忽略 `async: true`、退回同步响应，不创建任务或存储图片。 | `ImageGenerationLogPollingIntervalSeconds`、`controller.GetStatus`、`RelayImageGeneration`、任务 payload、日志设置表单、usage logs 轮询、locale files | `go test ./...`、`bun run typecheck`、目标 `oxlint`/`oxfmt`、`bun run build`、`bun run i18n:sync`、`git diff --check` |
| 2026-07-19 | 修复异步生图任务 ID 弹窗在列表自动轮询后自行关闭：将选中任务和弹窗生命周期从表格单元格提升到日志页面，桌面和移动端共用稳定弹窗。 | `usage-logs-table.tsx`、`image-generation-logs-columns.tsx`、`lib/columns.ts` | `bun run typecheck`、目标 `oxlint`/`oxfmt`、`git diff --check` |
| 2026-07-19 | 为 `/v1/images/generations` 增加 `async: true` 本地异步任务模式：提交即写生图日志并返回任务 ID，复用同步 Relay 后台执行，支持用户隔离轮询、受保护图片读取、`pending/processing/success/failed` 状态、脱敏响应 JSON；后台生图日志增加可点击任务 ID、状态弹窗和未完成任务自动刷新。 | `ImageRequest.async`、`ImageGenerationLog` 任务字段、`controller/image_generation_task.go`、`GET /v1/images/generations/:task_id*`、`GET /api/image-generation-logs/:id/task`、`image-generation-task-dialog.tsx`、locale files | `go test ./...`、`bun run typecheck`、目标 `oxlint`/`oxfmt`、`bun run build`、`bun run i18n:sync`、`git diff --check` |
| 2026-07-16 | 新增管理员/root 用户 IP 管理：历史登录 IP 多选封禁/解封、禁用用户事务联动封禁、普通用户登录/注册/会话/API Token 拦截，以及用户列表共享 IP/已封 IP 标记。 | `BlockedIP`、`GetUserLoginIPStats`、`ManageUser`、`AdminGetUserLoginIPs`、`AdminUpdateUserLoginIPs`、`middleware/auth.go`、用户详情/列表、locale files | `go test ./...`、`bun run typecheck`、`bun run build`、`oxfmt --check`、`bun run i18n:sync`、`git diff --check`；目标组件 lint 的 5 项既有问题单独记录 |
| 2026-07-16 | 用户详情在最近登录时间下显示未登录天数；按当前时间与 `last_login_at` 的完整 24 小时间隔动态计算，无登录记录时显示“从未登录”。 | `user-detail-dialog.tsx`、locale files | `bun run typecheck`、`oxfmt --check`、`bun run i18n:sync`、`git diff --check`；目标组件 lint 仅命中 5 项既有规则问题 |
| 2026-07-16 | 用户登录成功时记录最近登录 IP，并在管理员/root 用户详情基本信息中展示；统一登录收口覆盖密码、2FA、OAuth、微信、Telegram 和 Passkey。 | `User.last_login_ip`、`model.UpdateUserLastLogin`、`controller.setupLogin`、`user-detail-dialog.tsx` | `go test ./...`、`bun run typecheck`、`bun run i18n:sync`、`git diff --check`；目标组件 lint 仅命中 5 项既有规则问题 |
| 2026-07-14 | 新增 Docker 线上部署手册，记录本地构建/推送、服务器更新、数据备份、安全注意事项、回滚和常见故障处理。 | `docker线上部署.md` | 文件存在检查、Compose 参数核对、`git diff --check` |
| 2026-07-13 | 移除 API 密钥页端点列表上方重复的“自定义端点”标题，保留复制说明、端点条和悬停介绍。 | `keys/components/custom-endpoints.tsx` | `bun run typecheck`、目标 lint、`git diff --check` |
| 2026-07-13 | 新增自定义端点配置：管理员可在控制台内容/API 地址中维护名称、URL 和介绍；API 密钥页展示可点击复制的端点，悬停显示介绍。 | `console_setting.custom_endpoints`、`CustomEndpointsSection`、`CustomEndpoints`、`/api/status.custom_endpoints` | `go test ./...`、`bun run typecheck`、目标 lint、`bun run i18n:sync`、`git diff --check` |
| 2026-07-13 | 重做公告自动弹窗 UI：采用通知图标、未读标记、标题/时间头部、正文强调线和标记已读操作区；多条弹窗公告逐条标记已读并切换，关闭按钮不自动标记。 | `announcement-popup.tsx`、locale files | `bun run typecheck`、目标 lint、`bun run i18n:sync`、`git diff --check` |
| 2026-07-13 | 修复公告编辑器下拉框显示内部枚举值的问题；状态、通知方式、条件类型和运算符通过 Select items 映射显示本地化标签，并补齐草稿、归档、条件翻译。 | `announcements-section.tsx`、locale files | `bun run typecheck`、目标 lint、`bun run i18n:sync`、`git diff --check` |
| 2026-07-13 | 将公告升级为单条发布策略：支持草稿/展示中/已归档、静默/弹窗、起止时间、所有用户或 OR-AND 条件；条件支持订阅套餐和余额。新增当前用户公告过滤接口，通知中心与弹窗只展示命中公告。 | `Announcement`/`AnnouncementConditionGroup`、`GET /api/announcements`、`AnnouncementsSection`、`AnnouncementPopup`、`useNotifications` | `go test ./...`、`bun run typecheck`、目标 lint、`bun run i18n:sync`、`git diff --check` |
| 2026-07-13 | 客服弹窗中的 QQ 联系方式支持点击后通过桌面 QQ 注册的 `tencent://message` 协议唤起本机 QQ 并打开对应聊天；保留独立复制按钮。 | `support-contact-button.tsx` | `bun run typecheck`、目标 lint、`git diff --check` |
| 2026-07-12 | 将管理员系统设置权限扩展到全部 41 个二级菜单；增加菜单/路由过滤、`/api/option` 按页面过滤及配置键归属校验，并保护自定义 OAuth、性能、日志、支付、渠道亲和与价格同步专用接口。 | `service/authz/resources_system_settings.go`、`controller/system_settings_access.go`、`controller/option.go`、`router/api-router.go`、系统设置 permissions/routes/sidebar/settings API、管理员权限编辑器 | `go test ./...`、`bun run typecheck`、目标 lint、`bun run i18n:sync`、`git diff --check` |
| 2026-07-12 | 新增概览页联系客服弹窗、客服联系方式后台配置，以及 root 可分配给管理员的客服设置权限；QQ、微信和手机使用对应类型图标。 | `SupportContacts`、`GET/PUT /api/support-contacts`、`system_settings.content.support`、`support-contacts-section.tsx`、`support-contact-button.tsx`、系统设置路由/侧边栏、locale files | `go test ./...`、`bun run typecheck`、目标 lint、`bun run i18n:sync`、`git diff --check` |
| 2026-07-12 | 生图图片预览增加逐图下载、JSON 数据展示和 JSON 文件下载；桌面/移动端列表展示后端测量的请求总耗时。 | `image-generation-preview-dialog.tsx`、`image-generation-logs-columns.tsx`、`usage-logs-mobile-card.tsx`、locale files | `bun run typecheck`、`bun run i18n:sync`、`git diff --check` |
| 2026-07-12 | 修复生图日志开关假开启：日志设置保存时明确提交全部字段；增加聊天/Responses `image_generation_call.result` 捕获、去重、文件落盘和日志写入。 | `log-settings-section.tsx`、`ResponsesOutput.result`、`service/image_generation_log.go`、OpenAI Responses/Chat 转换处理器 | `go test ./service ./relay/...`、`bun run typecheck`、`git diff --check` |
| 2026-07-12 | 优化钱包页响应式布局与套餐卡片，展示套餐及当前订阅的生图日志查看范围。 | `wallet/index.tsx`、`wallet-stats-card.tsx`、`subscription-plans-card.tsx`、locale files | `bun run typecheck`、`bun run i18n:sync`、`git diff --check` |
| 2026-07-12 | 新增订阅套餐生图日志权限和最近 N 条限制；0 为全部，多有效订阅取最大权益，列表及图片读取均强制鉴权。 | `SubscriptionPlan`、`UserSubscription`、`GetUserImageGenerationLogAccess`、`GET /api/image-generation-logs*`、订阅套餐编辑抽屉、locale files | `go test ./model ./controller`、`bun run typecheck`、`bun run i18n:sync`、`git diff --check` |
| 2026-07-12 | 新增同步生图日志：root 开关、保留天数、base64 文件落盘、用户隔离查询/图片接口，以及任务日志中的生图日志和图片预览。 | `ImageGenerationLog`、`ImageGenerationLogEnabled`、`ImageGenerationLogRetentionDays`、`/api/image-generation-logs*`、Relay 图片适配器、`web/default/src/features/usage-logs/*`、日志维护设置、locale files | `go test ./model ./service ./controller ./relay/...`、`bun run typecheck`、`bun run i18n:sync`、`git diff --check` |
| 2026-07-12 | 在后台用户详情基本信息区增加有效订阅摘要。 | `web/default/src/features/users/components/user-detail-dialog.tsx`、`DEVELOPMENT.md`；复用 `GET /api/subscription/admin/users/:id/subscriptions` 和 `GET /api/subscription/admin/plans` | `bun run typecheck`、`bun run i18n:sync`、Rsbuild 热更新编译、`git diff --check` |
| 2026-07-12 | 优化代理详情和用户详情弹窗 UI，增加身份摘要、关键指标带、加载骨架、紧凑信息网格及移动端响应式表格。 | `web/default/src/features/users/components/agent-detail-dialog.tsx`、`user-detail-dialog.tsx`、`DEVELOPMENT.md` | `bun run typecheck`、Rsbuild 热更新编译、`git diff --check` |
| 2026-07-11 | 将 `DEVELOPMENT.md` 翻译为中文。 | `DEVELOPMENT.md` | `git diff --check` |
| 2026-07-11 | 创建 `DEVELOPMENT.md` 作为强制项目开发记录。 | `DEVELOPMENT.md` | 文档变更 |
| 2026-07-11 | 在计费设置中增加签到奖励额度预览。 | `web/default/src/features/system-settings/general/checkin-settings-section.tsx`、locale files | `bun run typecheck`、`bun run i18n:sync`、`git diff --check` |
| 2026-07-11 | 增强兑换码按 code/key 和生成者搜索。 | `model/redemption.go`、`redemptions-table.tsx`、locale files | `go test ./model ./controller`、`bun run typecheck`、`bun run i18n:sync` |
| 2026-07-11 | 增加后台用户详情弹窗和 token 总量 API。 | `controller/user.go`、`model/log.go`、`router/api-router.go`、`web/default/src/features/users/*` | `go test ./...`、`bun run typecheck`、`bun run i18n:sync` |
| 2026-07-11 | 增加 500 页面错误反馈提交/后台查看流程。 | `model/error_report.go`、`controller/error_report.go`、`router/api-router.go`、`web/default/src/features/error-reports/*`、routes、sidebar config | `go test ./...`、`bun run typecheck`、`bun run i18n:sync` |
| 2026-07-11 | 添加/迭代代理兑换码退款日志和兑换码移动端操作。 | `model/redemption.go`、`controller/redemption.go`、`web/default/src/features/redemption-codes/*`、usage logs | 实现过程中执行 Go/前端检查 |
| 2026-07-10 | 为 Julong 调整 Docker 部署命名/镜像。 | `docker-compose.yml`、README updates | Docker build/push/deploy 手动检查 |

## 后续开发 Checklist

每次完成改动前必须检查：

- [ ] 如果 API、组件、模型、配置行为发生变化，已更新 `DEVELOPMENT.md`。
- [ ] 已在变更日志中新增日期、文件/API/模型和验证记录。
- [ ] 前端 UI 文案已通过脚本更新所有语言，并运行 `bun run i18n:sync`。
- [ ] 后端改动已运行 `go test ./...`，或至少运行目标包测试并说明未全量测试原因。
- [ ] 前端改动已运行 `bun run typecheck`。
- [ ] 已运行 `git diff --check`。
- [ ] 如果 Go 路由/controller/model 行为变更且用户正在本地调试，已重启本地后端。
