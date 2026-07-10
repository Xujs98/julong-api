# 矩龙-API / julong-api

矩龙-API 是基于 New API 二次开发的 AI API 网关项目，用于统一管理模型渠道、用户、令牌、计费、兑换码和代理业务。

本仓库是二开版本，已脱离官方 Docker 镜像发布流程。部署时请使用本仓库源码自行构建镜像，不要使用 `calciumion/new-api:latest`。

## 二开说明

当前版本包含以下定制内容：

- 项目品牌调整为 `矩龙-API / julong-api`
- Docker Compose 使用本地源码构建 `julong-api:local`
- 支持按次计费模型的订阅抵扣开关
- 管理后台增加代理身份、代理折扣和代理充值链接配置
- 代理可生成兑换码，费用按 `兑换码额度 * 折扣 * 数量` 从代理余额扣除
- 代理兑换码扣费只扣余额，不扣订阅额度
- 代理可查看自己邀请的用户、兑换码和基础消费/订阅信息
- 管理员可查看用户所属代理和代理详情
- 代理邀请用户的钱包兑换码购买链接可使用代理自己的充值链接

## 技术栈

- 后端：Go
- 前端：React / TypeScript / Rsbuild
- 包管理：Bun
- 数据库：PostgreSQL，也可按项目配置切换 MySQL / SQLite
- 缓存：Redis
- 部署：Docker / Docker Compose

## 本地 Docker 部署

首次构建并启动：

```bash
docker compose build --no-cache
docker compose up -d
```

查看状态：

```bash
docker compose ps
```

查看日志：

```bash
docker compose logs -f julong-api
```

访问：

```text
http://localhost:3000
```

## 服务器部署

推荐把源码同步到服务器后，在服务器本地构建：

```bash
cd /opt/julong-api
docker compose build --no-cache
docker compose up -d
```

以后更新代码后执行：

```bash
cd /opt/julong-api
git pull
docker compose build --no-cache
docker compose up -d
```

不要执行：

```bash
docker compose pull
```

本项目默认使用本地源码构建，`pull` 远程镜像不会包含你的二开代码。

也不要随便执行：

```bash
docker compose down -v
```

`-v` 会删除数据库卷，可能导致数据丢失。

## 同步到服务器时建议排除

如果不用 Git，而是用 `rsync` 上传源码，建议排除本地依赖和运行数据：

```bash
rsync -avz --progress \
  --exclude 'web/node_modules' \
  --exclude 'web/default/dist' \
  --exclude 'web/classic/dist' \
  --exclude 'node_modules' \
  --exclude 'logs' \
  --exclude 'tmp' \
  --exclude 'data' \
  --exclude '.git' \
  ./ root@服务器IP:/opt/julong-api/
```

## 维护说明

- 修改代码后需要重新构建镜像：

```bash
docker compose build --no-cache
docker compose up -d
```

- 数据库字段会在后端启动时自动迁移。
- 生产环境请修改默认数据库密码、Redis 密码和会话密钥。
- 对外提供服务前，请确认域名、HTTPS、支付、内容安全和上游 API 授权配置。

## 开源来源

本项目基于 New API 二次开发。请在遵守原项目许可证和相关第三方许可证的前提下使用、修改和部署。
