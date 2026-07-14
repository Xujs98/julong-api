# 矩龙-API Docker 线上部署

本文档适用于当前仓库的生产部署与日常更新。

## 当前部署参数

| 项目 | 当前值 |
| --- | --- |
| GitHub 仓库 | `https://github.com/Xujs98/julong-api.git` |
| Docker Hub 镜像 | `qq1371446705/julong-api:latest` |
| Compose 应用服务 | `julong-api` |
| 应用容器名 | `julong-api` |
| Redis 容器名 | `julong-api-redis` |
| PostgreSQL 容器名 | `julong-api-postgres` |
| 宿主机端口 | `3388` |
| 容器应用端口 | `3000` |
| 服务器项目目录 | `/root/julong-api` |
| PostgreSQL 数据卷 | `pg_data` |

## 重要说明

服务器的 `docker-compose.yml` 使用 Docker Hub 镜像：

```yaml
image: qq1371446705/julong-api:latest
```

因此，代码更新后仅在服务器执行 `git pull` 不会更新正在运行的程序。完整流程是：

1. 本地提交并推送 GitHub。
2. 本地构建 Linux Docker 镜像。
3. 本地推送镜像到 Docker Hub。
4. 服务器拉取 GitHub 最新配置。
5. 服务器拉取最新 Docker 镜像。
6. 只重建 `julong-api` 应用容器。

## 一、本地构建并推送镜像

### 1. 确认代码已提交

```bash
cd /Users/xujs/.docker_dir/new-api

git status
git log -1 --oneline
git push origin main
```

### 2. 启动并检查 Docker Desktop

```bash
docker info
```

如果出现无法连接 `docker.sock`，先启动 Docker Desktop，等待 Docker Engine 就绪后再执行。

### 3. 登录 Docker Hub

首次使用或登录过期时执行：

```bash
docker login
```

Docker Hub 用户名为 `qq1371446705`。

### 4. 构建服务器镜像

当前服务器使用 Linux AMD64，Mac 本地构建时必须指定平台：

```bash
cd /Users/xujs/.docker_dir/new-api

docker build --platform linux/amd64 \
  -t qq1371446705/julong-api:latest .
```

不建议日常更新使用 `--no-cache`，它会明显增加构建时间和资源占用。只有确认缓存异常时才使用：

```bash
docker build --no-cache --platform linux/amd64 \
  -t qq1371446705/julong-api:latest .
```

### 5. 检查本地镜像

```bash
docker image inspect qq1371446705/julong-api:latest \
  --format '{{.Id}} {{.Architecture}} {{.Os}}'
```

预期架构为 `amd64`，系统为 `linux`。

### 6. 推送 Docker Hub

```bash
docker push qq1371446705/julong-api:latest
```

出现以下内容表示推送成功：

```text
latest: digest: sha256:...
```

## 二、服务器拉取并更新

登录服务器：

```bash
ssh root@38.246.244.17
```

进入项目目录：

```bash
cd /root/julong-api
```

### 1. 建议先备份数据库

```bash
mkdir -p /root/julong-api/backups

docker exec julong-api-postgres pg_dump \
  -U root -d julong-api \
  > /root/julong-api/backups/julong-api-$(date +%Y%m%d-%H%M%S).sql
```

确认备份文件不是空文件：

```bash
ls -lh /root/julong-api/backups
```

### 2. 拉取 GitHub 最新代码和 Compose 配置

```bash
git pull origin main
```

如果只修改了程序代码而没有修改 Compose，这一步仍建议执行，确保服务器仓库与 GitHub 一致。

### 3. 拉取最新应用镜像

```bash
docker compose pull julong-api
```

### 4. 只重建应用容器

```bash
docker compose up -d --no-deps --force-recreate julong-api
```

参数说明：

- `-d`：后台运行。
- `--no-deps`：不重启 PostgreSQL 和 Redis。
- `--force-recreate`：强制使用刚拉取的镜像创建新应用容器。
- `julong-api`：只操作应用服务。

这条命令不会删除 PostgreSQL 数据卷。

### 5. 检查部署结果

```bash
docker compose ps
docker compose logs --tail=100 julong-api
curl -fsS http://127.0.0.1:3388/api/status
```

健康状态应显示 `julong-api` 为 `Up` 或 `healthy`，状态接口应返回：

```json
{"success":true}
```

线上访问地址：

```text
https://api.julongkj.top
```

## 三、以后日常更新的标准命令

### 本地 Mac

```bash
cd /Users/xujs/.docker_dir/new-api

git push origin main

docker build --platform linux/amd64 \
  -t qq1371446705/julong-api:latest .

docker push qq1371446705/julong-api:latest
```

### 服务器

```bash
cd /root/julong-api

git pull origin main
docker compose pull julong-api
docker compose up -d --no-deps --force-recreate julong-api
docker compose ps
docker compose logs --tail=100 julong-api
```

## 四、数据安全

当前持久化数据包括：

- PostgreSQL：Docker 命名卷 `pg_data`。
- 生图日志等应用文件：`./data:/data`。
- 应用日志：`./logs:/app/logs`。

正常执行以下命令不会删除数据：

```bash
docker compose pull julong-api
docker compose up -d --no-deps --force-recreate julong-api
```

不要在生产服务器执行：

```bash
docker compose down -v
docker volume rm julong-api_pg_data
```

`-v` 会删除 Compose 数据卷，可能造成数据库永久丢失。

## 五、回滚

长期维护不建议只使用 `latest`。每次发布可同时创建版本标签：

```bash
VERSION=2026.07.14-1

docker build --platform linux/amd64 \
  -t qq1371446705/julong-api:$VERSION \
  -t qq1371446705/julong-api:latest .

docker push qq1371446705/julong-api:$VERSION
docker push qq1371446705/julong-api:latest
```

需要回滚时，在服务器将 `docker-compose.yml` 中镜像临时改为目标版本：

```yaml
image: qq1371446705/julong-api:2026.07.14-1
```

然后执行：

```bash
docker compose pull julong-api
docker compose up -d --no-deps --force-recreate julong-api
```

回滚程序镜像不会自动回滚数据库结构。涉及数据库结构变更时，部署前必须先备份数据库。

## 六、常见问题

### Docker daemon 未启动

```text
failed to connect to the docker API
```

处理：启动 Docker Desktop，然后执行 `docker info`。

### Redis 容器名称冲突

```text
The container name "/redis" is already in use
```

当前 Compose 已使用 `julong-api-redis`。服务器先执行 `git pull origin main`，确认使用仓库最新的 `docker-compose.yml`。

### 应用容器反复重启

```bash
docker compose ps
docker compose logs --tail=200 julong-api
docker compose logs --tail=100 postgres
docker compose logs --tail=100 redis
```

重点检查：

- `SQL_DSN` 的用户名、密码、数据库名是否一致。
- `REDIS_CONN_STRING` 密码是否与 Redis 启动参数一致。
- 数据库和 Redis 是否已正常启动。
- 宿主机 `3388` 端口是否被其他程序占用。

### 拉取镜像后页面仍是旧版本

强制重建应用容器：

```bash
docker compose pull julong-api
docker compose up -d --no-deps --force-recreate julong-api
```

然后清理浏览器缓存或使用无痕窗口检查。

### 查看正在使用的镜像

```bash
docker inspect julong-api \
  --format '{{.Config.Image}} {{.Image}}'
```

## 七、生产安全检查

当前 `docker-compose.yml` 中仍有示例密码 `123456`。正式生产环境必须修改：

- PostgreSQL 的 `POSTGRES_PASSWORD`。
- 应用 `SQL_DSN` 中的数据库密码。
- Redis 的 `--requirepass`。
- 应用 `REDIS_CONN_STRING` 中的 Redis 密码。
- 设置随机且足够长的 `SESSION_SECRET`。
- HTTPS 环境启用 `SESSION_COOKIE_SECURE=true`。
- 配置 `SESSION_COOKIE_TRUSTED_URL=https://api.julongkj.top`。

修改密码时，应用连接字符串和对应服务密码必须保持一致。
