# 矩龙项目进度记录

> 分批记录当前状态、验证证据和未完成事项，供新对话接续工作。
> 详细 API、路由、配置和数据模型以 [DEVELOPMENT.md](./DEVELOPMENT.md) 为技术登记入口。
> 代码与实际命令输出优先于旧文档、历史回复和截图；“代码已写”“本地可用”“已提交”“已上线”必须分开。

最后更新：2026-10-15 07:45:00（Asia/Shanghai）
项目根目录：`/Users/xujs/.docker_dir/new-api`

## 1. 新对话接续入口（必须先读）

1. 先读“当前状态”、最新批次、当前任务对应的历史批次、未完成项和验证记录。
2. 读取根目录及相关子目录 AGENTS.md。触及计费、认证、数据库、插件或前端时继续读取必读规则，不以本文替代。
3. 用当前代码、git status、分支及 MERGE_HEAD 核对快照；不得依文档覆盖现有更改。
4. 优先接续未完成工作。已完成项被用户反馈回归时重新打开原批次，保留旧结果的时间与局限。
5. 每批开始、重要进展、阻塞、移交或交付前更新本文；同步 DEVELOPMENT.md 受影响的技术清单和变更日志。
6. 只记录可核实事实。未重新验证的历史结果注明“历史验证”；被 Skip 的测试不能计作通过。

禁止写入 API Key、Cookie、session secret、数据库密码、私人数据和原始 Base64 大文件。

## 2. 当前状态快照

| 项目 | 状态（截至最后更新时间） | 依据与边界 |
| --- | --- | --- |
| 当前分支 / HEAD | `codex/upstream-sync-20261013` / `efa2da356` | 本地分支跟踪 `origin/codex/upstream-sync-20261013`；仅剩未跟踪 `web/pnpm-lock.yaml`，不属于本批功能 |
| 上游整合 | 已包含在待合并分支，当前可快进至主分支 | `f99b63cfc` parents=`7d3fba2f4` + `ef60ef27f`；`origin/main...HEAD = 0 374`，主分支是当前 HEAD 祖先，无冲突路径；尚未推送到 main |
| 用户 GitHub 仓库 | origin = https://github.com/Xujs98/julong-api.git | upstream = https://github.com/QuantumNous/new-api.git，仅跟踪上游；不得把用户改动推到 upstream |
| 本地服务 | 最新前端已重新构建并嵌入，PID49923监听3000，健康检查通过 | `GET /api/status success=true`；路由 `/models/document-templates` 返回首页回退及 `/static/js/index.5f11f9b712.js`；页面需管理员登录后验证真实管理交互 |
| 模型广场按秒/按次价格 | 当前代码已实现，历史交付有验证记录 | 两项都有价格同时显示；只配一项只显示该项；模型名及分辨率价格区规则一致 |
| 视频输入素材 | 已完成本地实现/定向回归、三库复验、浏览器预览和视频直链、最终重启 | SQLite3.50.4/MySQL5.7.44/PostgreSQL16.15最终矩阵8/8无Skip；图片及嵌入视频/音频正常、素材视频直链可播放；Safari/原生音频直链/线上公网未实测 |
| 成品列阴影与滚轮 | 已完成（本地）；17项通过，Chromium已验证并激活；Safari待人工复核 | 独立复核10/10，主任务4文件17/17；Chromium明暗阴影、全列固定、滚轮/splitHeader已验；首页资源与dist一致，Safari无连接未实测 |
| 项目进度入口 | 已完成（文档本地） | 本文 + AGENTS.md 必读入口；持续按批次更新 |
| Git 提交 / 线上发布 | 分批提交已推至用户仓库功能分支；正在检查合并到 `main` | `origin/codex/upstream-sync-20261013` 指向 `efa2da356`；推送已成功。线上 `api.julongkj.top` 未部署/验证 |

## 3. 状态口径

- **待处理**：需求已登记，尚未实现。
- **进行中**：正在实现/修复，须说明当前步骤。
- **已实现 / 待验证**：代码存在，但必需测试、浏览器或数据库矩阵未完成。
- **已验证 / 待交付**：相关验证通过，仍需重启、文档或其他收尾。
- **已完成（本地）**：实现、必需验证、文档和必要重启均完成；不暗示提交/上线。
- **已提交 / 已发布**：另列 commit、remote/branch、时间和部署验证，不能凭旧聊天推断。
- **需补证 / 外部阻塞**：写清缺失条件和下一步，不得标完成。

## 4. 分批进度

### B08：模型文档模板（2026-10-15，已本地验证/待 Git 交付）

- 用户目标：模型管理上传 Markdown/Word 模板，列表显示上传时间、名字及修改/改名/删除；模型编辑端点绑定模板；模型广场详情 API 调用示例展示所选模板。
- 开始/接续：2026-10-15 05:20/05:27（Asia/Shanghai）。
- 当前状态：后端 CRUD/Markdown 与 DOCX 转换、管理端管理/模型绑定、模型广场安全展示均已实现。根据用户页面反馈，将“文档模板”标签移入与模型/供应商/部署相同的标签组末尾，并改为单行内部横滚布局。
- 约束：优先复用安全 Markdown/表格/弹窗等已有组件；模板不影响原模型计费、视频或代理功能。Word 支持格式及转换边界以实际实现和回归为准，不预先声称旧 `.doc` 可用。
- 规则：AGENTS.md 已补每批进度/本地提交/只推用户 origin、推送失败仅重试一次、二次失败留到下次任务规则。
- Git 范围：HEAD 为 `f99b63cfc`，MERGE_HEAD 为空；所有既有历史工作均为未提交工作区改动。用户明确授权“一并保存，分批提交并推送”，目标只可为 origin。
- 前端最终检查（本轮）：首轮模板管理/选择器/模型定价相关测试 3 文件 38/38；此次标签顺序/布局与管理页面回归 3 文件 14/14、`tsgo -b`、相关 lint/format 通过；最新 Rsbuild build 成功。环境缺 Bun 命令，使用 bundled Node 和仓库现有 node_modules。
- 后端定向复核（本轮）：模板 CRUD/解析与 PAT 路由测试通过；复核机仅 SQLite 3.50.4，无 DSN 的 MySQL/PostgreSQL 子项未运行。历史独立临时库矩阵 SQLite 3.50.4、MySQL 5.7.44、PostgreSQL 16.15 fresh/upgrade 6/6 通过，见 `/tmp/julong-documents-verify-20261015/{matrix.log,controller-matrix.log}`，不是本轮重跑。
- 安全修复：DOCX 非矩形表格先扩张造成内存放大、并发改名旧快照覆盖正文、scoped PAT 路由缺权限声明已修复并有定向回归。
- 全量 Go 检查限制：已有 `/tmp/julong-documents-full-go-tests.log` 记录 `go test ./...` 失败（涵盖认证/会话、Responses WebSocket、任务结算及固定计费数据库矩阵等），不得声称全量通过；定向测试结果单独报告。
- 后端定向验证（本轮）：`go test ./controller -run 'Test(ModelDocumentTemplate|GetModelDocumentation)' -count=1` 通过。
- 本地重启（本轮）：重新构建前端并嵌入 Go 二进制；旧PID15826已停止，最新PID49923监听3000，`GET /api/status success=true`，模型模板路径回退页面返回新主包哈希 `index.5f11f9b712.js` 和CSS。未提供管理员登录态，因此没有假称真实已登录交互已验。
- 页面反馈回归：原标签列表允许换行且文档模板顺序落在供应商与部署之间；现移到部署之后，同一 TabsList 设置单行与 `overflow-x-auto`，避免挤出灰色标签组。顺序契约回归新增 `section-registry.test.ts`。
- Git交付：功能与文档提交已推送至 `origin/codex/upstream-sync-20261013`，含 `298b2cf33`、`dfb2b1425`、`734a26746`、`efa2da356`。本轮远端比较 `origin/main...HEAD = 0 374`，主分支是当前 HEAD 祖先；`git diff --check origin/main...HEAD` 通过，分支可 fast-forward 到 main，尚未执行主分支推送。
- `web/pnpm-lock.yaml` 为单独未跟踪文件；项目使用 `web/bun.lock`，该额外文件排除在合并外。
- api.julongkj.top 未部署；全量 `go test ./...` 有已知失败记录 `/tmp/julong-documents-full-go-tests.log`，不包装成全量通过。
- 下一步：复核远端 `main` 未移动后，在用户授权下 fast-forward 推送至 `origin/main`，验证远端 main SHA 与本地 HEAD 一致。

### B01：矩龙原功能与上游整合基线

- 目标：在矩龙现有项目上合并上游，不替换/删除矩龙定制功能。
- 保留清单：代理系统、特殊计费倍率/收益审计、风控/IP/设备封禁、邮件提醒、发票、运营账本、异步生图、日志导出、管理员细粒度权限。
- 证据：DEVELOPMENT.md 各功能与上游整合章节，工作区合并目标 ef60ef27f36e418143fa4abf0c9025ba6f17fba7。
- 状态：整合代码存在；Git 合并提交未闭环，未重新验收全部保护功能。
- 历史验证：技术文档留有 Go 测试、typecheck/build 等记录，本轮未重跑全量，不扩大为全站无回归或商用级结论。
- 下一步：提交/发布前核对保护清单、验证矩阵和工作区范围；有授权时仅向 Xujs98/julong-api 推送。

### B02：后台页面、API 密钥滚动与自助发票扩展

- 目标：分组定价页面修复；密钥列表过多时可滚动；发票增加信息、默认注册邮箱且可修改、限制可申请充值来源。
- 证据：keys、invoices、system-settings 及 controller/model/invoice 等实现与 DEVELOPMENT.md 历史记录。
- 状态：历史实现/交付事项，本轮不重复修改，亦未重验全部页面。出现回归时以真实页面/接口结果重新打开本批。
- 下一步：涉及发票 schema 的后续变更继续遵循三数据库真实验证，编译不能替代。

### B03：算力视频渠道、生成与任务日志

- 目标：适配算力 AI；渠道按模型配置；查看轮询结果/请求体、普通用户隔离、成品预览。
- 实现：plugins/tasks/suanliai-video、视频模型配置、/v1/videos 提交/轮询/内容接口、任务结果/请求弹窗、视频 URL 本站代理、视频生成使用指南.md。
- 任务日志：成功“查看视频”、失败原因、无内容占位、类型彩色标签、ID 单行省略/悬停、JSON 格式化/高亮；普通用户隐藏平台且只读本人任务。
- 历史验证：DEVELOPMENT.md 的 2026-10-14/15 任务日志章节，相关测试/构建和部分浏览器检查；本轮审计未新建付费视频任务。
- 本轮验证：用户切换VPN DNS后 suanliai.top 解析真实IP，既有成品Range请求HTTP206/video/mp4、0-1023/11173528、MP4头正确；Chromium嵌入视频及直接签名URL原生页面均readyState=4、1280×720、14.08s、无媒体错误，点击播放后paused=false且currentTime增长。使用已有任务，未新建付费视频任务，也未关闭SSRF防护。
- 验证边界：本地既有视频的完整浏览器加载/播放及直接链接已验证，不代替线上和第三方工具端到端验证。旧fake-IP（198.18.x.x）阻塞记录作为历史保留，不再作为当前失败结论。
- 约束：不新增生成视频自动过期/清理；Base64 **输入素材** 的临时清理与视频成品分开。

### B04：输入素材托管与本站线上接口文档

- 目标：Base64 图片/视频/音频暂存本站并生成公网 URL，默认 1 天过期清文件/记录；用户素材预览/复制，管理员查看全站/删除；用户公网 URL 保持原地址。
- 实现：model/media_asset.go、service/media_asset.go、controller/media_asset.go、usage-logs/components/media-assets-table.tsx、运维设置与系统任务。
- 配置：MediaAssetRetentionDays、MEDIA_ASSET_STORAGE_DIR、TaskPublicAddress（为空回退 ServerAddress）；预期线上根地址 https://api.julongkj.top。本地没有保存公网地址配置，目前仍回退 localhost，不以此宣称能供上游公网访问，也不静默修改部署设置。
- 接口：GET /api/media-assets；DELETE /api/media-assets/:asset_id；GET/HEAD /api/media-assets/:asset_id/content。合同/权限见 DEVELOPMENT.md。
- 本站文档已完成：根 `全站通用视频接口文档.md` 根据用户源文档和当前实现核对，本站域名 api.julongkj.top，含视频流程、face默认、Base64素材/清理、签名成品与Bearer内容地址、Python下载及部署条件；`视频生成使用指南.md` 已链接。用户原路径 /Users/xujs/Downloads/全站通用视频接口文档.md 亦有先前改造结果。修正早期仅查仓库指南而将源文档标“未完成”的判断；文档完成不等于已上线。
- 本轮独立报告：/tmp/julong-media-verify-20261015/REPORT.md，验证代理未修改仓库业务代码、现有数据库或用户配置；使用临时夹具/专用测试库。
- 后端验证：`go test ./service -run 'Test(PersistTaskMedia|MediaAssetSchemaCompatibility|ValidateUserMediaURL)' -count=1 -v` 初次明确有2个DSN未配置Skip；后续真实DSN重跑 `go test ./service -run TestMediaAssetSchemaCompatibility -count=1 -v`，MySQL/PostgreSQL两子项实测通过、无Skip，不能将两次混写。日志 /tmp/julong-media-verify-20261015.log 与 /tmp/julong-media-schema-real-20261015.log。
- 前端验证：`cd web && bun run test src/features/usage-logs/components/__tests__/media-assets.test.tsx`，2/2；日志 /tmp/julong-media-ui-20261015.log。
- 真实宿主数据库矩阵：`python3 /tmp/julong-media-verify-20261015/run_matrix.py`（内部 go test 临时 media_matrix_test.go -run TestMediaAssetHostMatrix -count=1 -v），8/8无Skip。SQLite 3.50.4 新库/代表性旧库2场景；MySQL 5.7.44 新库/升级/独立日志库3场景；PostgreSQL 16.15 同3场景。每场景 model.InitDB/InitLogDB 两次、schema稳定、asset_id唯一约束有效、旧用户/成功任务/Option不变。旧库仅代表素材加入前的用户/任务/配置，不扩大为确切最新发布版全量迁移认证；矩阵 /tmp/julong-media-verify-20261015/matrix.log。
- 核心行为实测：图片/视频/音频Base64文件内容正确、默认+86400秒；用户只读本人/管理员全站；普通用户删除403；管理员删除清文件/记录；过期410、清理后404；公网URL原样保存、不落盘/不过期；直接控制器GET/HEAD/Range合同通过；本地匿名列表401。不是浏览器真实解码或付费上游端到端。
- 发现并修复：全局API gzip曾对素材内容二次压缩，破坏Range原始字节和HEAD空body。router/api-router.go现排除 /api/media-assets/[^/]+/content，列表JSON仍压缩；controller/media_asset.go的CSP允许同源media播放。失败证据 gzip-before.log；临时夹具试修 gzip-after.log；仓库新增 router/media_asset_router_test.go 的真实路由GET/HEAD回归已通过，日志 /Users/xujs/.codex/tmp/julong-api-runtime/media-range-after-20261015.log；涉及发票/任务路由注册的定向回归亦通过。
- 最终续修：sandbox的不透明源使self媒体直链停在readyState0，已在素材控制器与生成视频代理改为sandbox allow-same-origin，继续default-src none且不允许脚本；真实仓库路由隔离视频直链320×180/3s/readyState4/error=null，点击paused=false且currentTime增长，图片和嵌入视频/音频正常。Safari未实测，音频原生直链导航被浏览器控制端拦截，不算通过；HTTP音频字节/Range和嵌入解码已通过。
- 字段误判已修：明确媒体字段/别名，不再contains匹配；普通image_size/video_quality、豆包sequential_image_generation、xAI voice_id及素材role保留；只向url继承媒体类型，首尾帧/camelCase/type素材对象/文件占位兼容。5项回归先4失败/1通过，修复后5/5。
- Base64限额已补：解码前核算编码长度/解码大小，50MiB每文件、128MiB剩余总预算，保留CRLF/Std/Raw；小边界/非法/空值、真实超50MiB、剩余总量、取消与既有失败回滚测试通过。没有证明宿主请求体限制可被绕过，亦不宣称全站资源安全认证。
- 最终验证：`go test ./service ./router ./controller -run 'Test(PersistTaskMedia|MediaDataURL|ValidateUserMediaURL|MediaAssetContent|ProxyTaskMedia|SetTaskRouter|InvoiceAPI)' -count=1 -v` 三包通过、无Skip，日志media-final-tests-20261015.log；`python3 /tmp/julong-media-verify-20261015/run_matrix_final.py`三库8/8再次通过、无Skip，证据matrix-final.log。
- 状态：本地已完成。最终Go构建、PID15826重启、健康/6资源hash/匿名列表401通过；既有生成视频Range206/1024字节MP4且新CSP已生效。未修改用户数据/计费/schema，未线上部署或新建付费任务。
- 下一步：生产配置任务公网地址和持久化目录后验证上游可访问；Safari/外部工具按可用环境补验。不得代删用户公网文件，不对生成视频成品加过期处理。

### B05：模型广场视频价格展示（2026-10-15）

- 目标：按秒金额/秒，按次金额/次；同时配置同时显示，仅一项就单独显示，分辨率价格区规则一致。
- 实现：pricing/components/model-card.tsx；复用任务价格解析/货币换算；保留分辨率及条件，不改变扣费。
- 回归：pricing/__tests__/task-price-display.test.tsx 及价格矩阵/条件测试。
- 状态：历史交付称本地完成；本轮确认代码及回归用例存在。
- 历史验证（DEVELOPMENT.md 2026-10-15）：3 个 Vitest 文件 90/90；typecheck、目标 lint/format、前端 build、根模块 go build、git diff --check；重启、/api/status、资源与定价配置检查。
- 边界：无此批提交/推送/上线证据；本轮未重复宣称全量通过。

### B06：成品列跨浏览器阴影与滚轮（2026-10-15，本轮）

- 目标：固定整个成品列、保留成功/失败/暂无内容；左侧明确层级；列内鼠标滚轮横向。
- 实现：复用共享 DataTableView 与 DataTablePage；新增 opt-in horizontalWheelColumnId，只在桌面任务成品列启用；统一表格与 splitHeader 都支持。原生非 passive wheel 监听避免 Chrome 无法拦截。列外、Ctrl 缩放、无溢出、已到边界保留原生滚动。
- 阴影：固定单元格外侧 14px 独立渐变伪元素 + 边界线，不再依赖 collapsed table 单元格 box-shadow；固定列 overflow-visible，静态文字仍由内层 TruncatedCell 截断，背景保持不透明。暗色阴影单独增强。
- 自动验证：`cd web && bun run test src/components/data-table/ src/features/usage-logs/components/__tests__/task-dialogs.test.tsx`，4 个文件 17/17，通过（04:33）；覆盖子按钮/表头、双方向、非成品列、Ctrl、无溢出/边界、line/page 单位、splitHeader 和 opt-out。首次缺陷用例失败后修复通过。
- 静态验证：`bun run typecheck`、8 个相关 TS/TSX 文件目标 oxlint、9 个文件 oxfmt 检查、定向 git diff --check 全部通过；不是全仓 lint 验收。
- 浏览器验证：CUA Chromium，临时 localhost:4319 夹具导入真实共享组件和最新生产 CSS，成功/失败/暂无内容均可见；成品列 x=831、right=991 在横滚前后不变。14px 明色 alpha=.18/暗色 alpha=.5 阴影肉眼可见；按钮上滚轮改变 scrollLeft，未到边界时 pageY 不变，splitHeader 同样生效。没有使用真实用户或模拟鉴权去访问生产数据。
- 浏览器边界：当前仅有 Codex Chromium 浏览器，Safari 没有可用控制连接，未宣称 Safari 真实验收；方案使用标准 CSS 渐变规避原差异。
- 构建/本地运行：`VITE_REACT_APP_VERSION=v1.0.0-julong.1 bun run build` 与根模块 `go build -ldflags ... -o .../new-api.next .` 成功；旧 PID 96689 退出，当前 PID 5915 监听 3000。`/api/status success=true`、版本正确，首页 6 个 JS/CSS 的 SHA256 与 web/dist 一致。
- 结果：本轮实现/测试/浏览器验证/构建/重启已闭环；Safari 人工复核仍列明确验证范围限制。本轮不改 API、数据库、实际价格或视频成品生命周期。
- Git/上线：未提交、未推送、未部署 api.julongkj.top。

### B07：项目进度文档与新对话入口（2026-10-15，本轮）

- 目标：分批记录，任意时刻可知进度；AGENTS.md 登记路径，Codex 新对话先读。
- 实现：根目录 PROJECT_PROGRESS.md；AGENTS.md 必读/续接规则与绝对路径；DEVELOPMENT.md 维护入口和事实修正。
- 原则：历史验收与本轮实测分开，整合/验证/提交/部署分开，不覆盖未完成事项。
- 状态：已完成（文档本地）；成品列、播放及素材证据持续在对应批次同步，AGENTS.md已要求新对话和上下文恢复先读。
- 本轮审计：git 分支/log/status、MERGE_HEAD、远端、未合并路径、素材测试 Skip、技术登记及健康检查；不修改业务代码。

## 5. 未完成事项与优先级

| 优先级 | 事项 | 下一步 | 禁止误报 |
| --- | --- | --- | --- |
| 待合并 | `codex/upstream-sync-20261013` → `origin/main` | 远端 main 是当前 HEAD 祖先（0/374），可 fast-forward；再次检查 main SHA 后推送；不 force-push |
| 待合并 | B08：模型文档模板及全部获授权历史工作 | 代码、验证、进度/技术文档及本地重启完成 | 后端定向通过不等于全量 Go 测试通过；历史三库矩阵不冒充本轮实测；未登录未声称已点测；本地运行不等于线上发布 |
| 已完成（本地） | B06：成品列阴影/滚轮 | Chromium真实组件验证和17项回归、构建重启均完成；Safari可用后人工复核 | 未宣称Safari已实测，旧实现不等于新增滚轮完成 |
| 已本地验证 | B03：成品直链播放 | Range/嵌入播放/直接签名URL已验；线上及第三方工具部署后再验 | 单个已有视频成功不代表全部上游永久可用 |
| 已完成（本地）/上线待验 | B04：素材/线上文档 | 最终三库8/8、回归、文档、浏览器图片/嵌入媒体/视频直链和重启已完成；部署配置公网地址后验上游访问 | Safari和原生音频直链未实测，不把本地成功等同线上付费端到端，不混淆素材与成品 |
| 交付纪律 | B07：持续记录 | 收尾更新状态/证据/下一步，新对话先读 | 不能只依赖历史聊天 |
| 提交/发布时 | B01：Git 合并及用户仓库 | 授权范围内核对验证、目标 remote | MERGE_HEAD 未完成不能称已提交，不推上游 |
| 持续 QA | 原代理/计费/发票/风控/账本保护功能 | 按影响补回归/环境矩阵 | 局部 UI 通过不等于全站商用级 |

## 6. 验证记录

每次登记实际时间（Asia/Shanghai）、工作目录、命令、通过/失败/Skip 数、环境版本和局限。历史验证可引用出处，不伪造输出。

本轮审计（2026-10-15 04:21:54）：

| 实际检查 | 结果 |
| --- | --- |
| git branch --show-current；git log -1 --format='%h %ad %s' --date=iso-strict | `codex/upstream-sync-20261013` / `f99b63cfc`；本地合并提交已创建 |
| git rev-parse -q --verify MERGE_HEAD | 空，未处于进行中合并 |
| git diff --name-only --diff-filter=U | 无未解决 Git 冲突路径 |
| git remote -v | origin Xujs98/julong-api；upstream QuantumNous/new-api |
| lsof -nP -iTCP:3000 -sTCP:LISTEN；ps -p 5915 -o pid,lstart,command | PID 5915 监听，2026-10-15 04:31:10 启动新本地后端 |
| GET http://localhost:3000/api/status；首页资源与 web/dist 字节比较 | success: true；6 个首页资源 200、逐字节一致；新阴影 CSS 已提供 |
| 读取 pinned-wheel-tests-20261015.log | 主任务 4 文件 17/17，实际命令/时间见 B06；构建与启动日志亦存在 |
| 读取 pinned-wheel-server-20261015.log | 04:33:05 既有成品 content HTTP 206，仅证明 Range 请求可达、不代替完整播放 |
| 独立素材验收报告与真实matrix.log | 8/8无Skip，实际版本/命令/行为/边界见B04；有单独真实DSN schema两子项与前端2/2 |
| media-range-after-20261015.log及当前路由 | 排除素材内容gzip后真实GET Range/HEAD回归通过；根Go构建和04:53:16新PID10797激活已验证，日志media-content-server-20261015.log |
| 独立复核：bun run test horizontal-wheel-scroll.test.tsx + task-dialogs.test.tsx | 2 个文件、10/10 项，退出码 0；依据 review_pinned_wheel 代理实际验证报告，未代替浏览器实测 |
| 文档入口/章节检查；git diff --check | 确认 7 个分批记录、AGENTS.md 绝对路径入口、DEVELOPMENT.md 维护入口与变更日志；通过 |

## 7. 后续批次模板（追加，保留历史）

### Bxx：任务名称（YYYY-MM-DD）

- 用户目标：
- 开始/最近更新（Asia/Shanghai）：
- 当前状态：
- 已完成：
- 未完成/阻塞：
- 文件/API/路由/配置：
- 实际验证命令/结果（失败、Skip 也记录）：
- 重启/健康检查：
- 提交/发布：未提交，或填写 commit、remote/branch、发布时间/线上验证。
- 下一步：

### 本轮检查点 — 2026-10-15 04:36:54（Asia/Shanghai）

B06 已完成本地实现/17项回归/构建/重启；B03 已有视频 Range、嵌入播放和直接URL均恢复；B04 本站文档已落地，素材三库证据继续核对。所有进度不暗示 Git 提交或线上更新。

### 客户端文档及播放检查点 — 2026-10-15 04:43:42

根线上视频文档已创建并核对当前协议；现有签名成品地址在 Chromium 中原生视频页面加载成功（readyState=4，1280×720，14.08s），嵌入播放器点击后播放时间增长。未修改价格、未新建付费任务、未推送/上线。

### 媒体修复激活检查点 — 2026-10-15 04:56:50（Asia/Shanghai）

真实路由回归先失败后通过；根Go构建成功，旧PID5915正常退出、新PID10797监听3000。状态成功/版本v1.0.0-julong.1，首页6资源SHA256一致，匿名素材列表401。B06 Chromium和B03已有视频播放均已验证；合成输入素材浏览器解码收尾中。没有修改用户数据/公网部署配置，没有Git提交推送。

### 素材续修检查点 — 2026-10-15 05:05:41（Asia/Shanghai）

隔离浏览器实测发现 sandbox 不透明源仍阻止输入素材原生直链播放，单有 media-src self 不足；对照 allow-same-origin 后相同3秒视频正常，正补回归与修复（保持禁脚本，不允许allow-scripts）。同时独立复核证实通用素材字段contains匹配误拦合法普通参数/voice_id/role，不只是理论风险，已分配限定service两文件修复。当前PID10797为上一阶段构建，最终补丁尚未再次激活；不得提前报告所有素材闭环。

### 最终本地交付检查点 — 2026-10-15 05:15:35（Asia/Shanghai）

B03已有成品正常播放，B04素材/本站文档及新发现的gzip/CSP/字段误判/解码前限额已修复并定向验证，B06阴影和滚轮已完成Chromium真实组件检查，B07进度入口已落地。最终service/router/controller定向测试三包通过无Skip、三数据库最终矩阵8/8无Skip。根Go构建成功，新PID15826于05:13:29启动，3000监听、状态成功、6资源hash一致、匿名素材列表401，既有成品Range206/MP4且最终CSP正确。Safari/音频原生直链/线上公网未实测，本轮无Git提交推送，无上游付费新任务。旧检查点保留，不可用旧PID或旧待验结论覆盖本段。

最终重启后Chromium再次重新加载既有成品直链：readyState4、1280×720、14.08秒、error=null。临时4319/4321/4322验证服务与代理创建的验证标签页已清理，仅保留3000项目服务；私密成品URL未写入进度或接口文档。

### 模型文档模板最终交付检查点 — 2026-10-15 06:55（Asia/Shanghai）

B08：MD/DOCX 后端、管理 UI、模型绑定与广场调用文档已实现。前端模板/价格测试 38/38，typecheck、目标 lint/格式检查、生产构建通过；后端定向模板、PAT 路由验证通过。历史 MySQL/PostgreSQL/SQLite 6/6 schema 矩阵不属于本轮重跑。全量 `go test ./...` 有明确失败记录。当前 PID15826 是旧二进制，重启/健康检查和 Git 交付尚待执行。用户授权分批提交全部既有历史工作，仅推 `origin`，不可推上游。

### 文档模板标签显示与 Git 交付接续 — 2026-10-15 08:30（Asia/Shanghai）

用户截图指出文档模板没有与“模型/供应商/部署”呈现为同组标签。源码此前已注册该段，但标签允许换行、顺序夹在供应商与部署之间；运行服务亦早于当前 `web/dist`。现顺序调整为模型→供应商→部署→文档模板，标签栏单行并在窄屏内部横向滚动。新加section顺序测试；模板管理/选择器/顺序共14/14，`tsgo -b`、目标lint/format通过，Rsbuild构建和根Go构建成功。本地PID49923监听3000，`/api/status success=true`，返回最新bundle哈希。未取得管理员登录态，未声明已登录交互点测。Git分批提交：`298b2cf33`、`dfb2b1425`、`734a26746`；push至origin分支尝试2次均因网络无法连接github.com:443失败，已按规则留待下一任务；未推upstream。
