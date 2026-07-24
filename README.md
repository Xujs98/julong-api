# 矩龙-API / julong-api

矩龙-API 是基于 New API 二次开发的 AI API 网关项目，用于统一管理模型渠道、用户、令牌、计费、兑换码和代理业务。

本仓库是独立维护的二开版本。生产部署使用 Julong 自有镜像 `qq1371446705/julong-api:latest`，不要使用上游 `calciumion/new-api` 镜像，否则不会包含本仓库的定制功能。

当前代码已于 2026-07-25 同步上游 `QuantumNous/new-api@84a79b68`，保留 Julong 业务功能并采用上游最新默认前端和认证架构。

## 二开说明

当前版本包含以下定制内容：

- 项目品牌调整为 `矩龙-API / julong-api`
- Docker Compose 使用 `qq1371446705/julong-api:latest`
- 支持按次计费模型的订阅抵扣开关
- 管理后台增加代理身份、代理折扣和代理充值链接配置
- 代理可生成兑换码，费用按 `兑换码额度 * 折扣 * 数量` 从代理余额扣除
- 代理兑换码扣费只扣余额，不扣订阅额度
- 代理可查看自己邀请的用户、兑换码和基础消费/订阅信息
- 管理员可查看用户所属代理和代理详情
- 代理邀请用户的钱包兑换码购买链接可使用代理自己的充值链接
- 支持兑换码生成者追踪、代理兑换限制和删除退款日志
- 支持用户详情、登录 IP 历史、共享 IP 标记与 IP 黑名单
- 支持错误反馈工单、公告弹窗、自定义 API 端点和客服联系方式
- 支持 root 为管理员分配系统设置页面权限
- 支持生图日志、订阅查看权益、异步任务轮询、图片读取白名单和 MinIO 生命周期清理

## 技术栈

- 后端：Go
- 前端：React / TypeScript / Rsbuild
- 包管理：Bun
- 数据库：PostgreSQL，也可按项目配置切换 MySQL / SQLite
- 缓存：Redis
- 部署：Docker / Docker Compose

## 本地开发

后端：

```bash
go run .
```

前端：

```bash
cd web
bun install
bun run dev
```

默认地址：后端 `http://localhost:3000`，前端 `http://localhost:5173`。

## Docker 发布

本地构建服务器使用的 Linux AMD64 镜像：

```bash
docker build --platform linux/amd64 \
  -t qq1371446705/julong-api:latest .
docker push qq1371446705/julong-api:latest
```

服务器更新：

```bash
cd /root/julong-api
git pull origin main
docker compose pull julong-api
docker compose up -d --no-deps --force-recreate julong-api
docker compose ps
docker compose logs --tail=100 julong-api
```

完整发布、备份、回滚和故障排查流程见 [`docker线上部署.md`](./docker线上部署.md)。不要在生产服务器执行：

```bash
docker compose down -v
```

`-v` 会删除数据库卷，可能导致数据丢失。

## 维护说明

- 每次修改 API、组件、模型或配置后同步更新 [`DEVELOPMENT.md`](./DEVELOPMENT.md)。
- 每次代码变更完成后提交并推送 GitHub；生产发布还需要重新构建并推送 Docker 镜像。
- 数据库字段会在后端启动时自动迁移。
- 生产环境请修改默认数据库密码、Redis 密码和会话密钥。
- 对外提供服务前，请确认域名、HTTPS、支付、内容安全和上游 API 授权配置。

## 开源来源

本项目基于 [QuantumNous/new-api](https://github.com/QuantumNous/new-api) 二次开发。请遵守原项目 AGPL-3.0 许可证和第三方许可证。
