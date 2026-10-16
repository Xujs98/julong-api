# 矩龙项目进度记录

> 分批记录当前状态、验证证据和未完成事项，供新对话接续工作。
> 详细 API、路由、配置和数据模型以 [DEVELOPMENT.md](./DEVELOPMENT.md) 为技术登记入口。
> 代码与实际命令输出优先于旧文档、历史回复和截图；“代码已写”“本地可用”“已提交”“已上线”必须分开。

最后更新：2026-10-16 19:56（Asia/Shanghai）
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
| 当前分支 / HEAD | main / B16功能043f1adc8已推送（实际HEAD以git为准） | 使用日志权限、在线统计与Root状态隐藏已验证；四项既有未跟踪内容保留 |
| 上游整合 | main已包含此前功能分支提交 | 实际git和远端对比已纠正旧“尚待合并”快照，保留历史批次记录 |
| 用户 GitHub 仓库 | origin = https://github.com/Xujs98/julong-api.git | upstream = https://github.com/QuantumNous/new-api.git，仅跟踪上游；不得把用户改动推到 upstream |
| 本地服务 | B16最新前端嵌入Go产物，PID38584监听3000 | /api/status成功、6个首页JS/CSS与web/dist一致、匿名活动接口401；真实管理员浏览器登录未实测 |
| 模型广场按秒/按次价格 | 当前代码已实现，历史交付有验证记录 | 两项都有价格同时显示；只配一项只显示该项；模型名及分辨率价格区规则一致 |
| 视频输入素材 | 已完成本地实现/定向回归、三库复验、浏览器预览和视频直链、最终重启 | SQLite3.50.4/MySQL5.7.44/PostgreSQL16.15最终矩阵8/8无Skip；图片及嵌入视频/音频正常、素材视频直链可播放；Safari/原生音频直链/线上公网未实测 |
| 成品列阴影与滚轮 | 已完成（本地）；17项通过，Chromium已验证并激活；Safari待人工复核 | 独立复核10/10，主任务4文件17/17；Chromium明暗阴影、全列固定、滚轮/splitHeader已验；首页资源与dist一致，Safari无连接未实测 |
| 管理员日志与Root隐私 | B14本地已完成，已推送origin/main | 四项独立跨账户权限、Root五类日志过滤和详情保护；三库查询/会话验证、前端54项及构建/重启通过；全量测试及真实登录浏览器未验收 |
| 禁用账户全站防护 | B15本地已完成，功能已推origin/main | 全站IP/可信身份拦截、自动追封、SSRF页独立入站名单、登录IP记录；三库与登录追加回归通过，人机验证需真实密钥，未上线 |
| 使用日志权限与在线统计 | B16已完成（本地），功能已推origin/main | 独立使用日志权限与Root隐私、日活/在线卡片、状态角标/筛选；三库、前端17项、构建/重启通过；Root不统计且不返回状态 |
| 项目进度入口 | 已完成（文档本地） | 本文 + AGENTS.md 必读入口；持续按批次更新 |
| Git 提交 / 线上发布 | B16功能043f1adc8已推origin/main；此回执另存 | 不推upstream；api.julongkj.top未部署 |

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
| 已核实 | 此前功能分支已在main | 本轮main及origin/main均为f0870f9d1，历史待合并项已关闭 | 不据旧快照重复合并或覆盖现有工作 |
| B09交付 | 素材管理UI/筛选与重建 | 本地验证/Go重启已完成，提交推送结果见B09最终检查点 | 不把全量侧栏旧失败/过滤Skip写成通过，不宣称生产部署 |
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

### B10：任务日志视频下载文件名改进（2026-10-15，已完成本地验证并交付）

- 用户目标：任务日志“查看视频”弹窗的下载按钮使用随机文件名，下载格式固定为 .mp4。
- 实现：视频成品卡片在自身生命周期内生成随机文件名并固定 .mp4 后缀，下载 URL 增加 download=video 标记；后端只在已确认视频类型且标记有效时覆盖上游 Content-Disposition，避免供应商文件名覆盖随机名称。图片、音频和普通文件下载行为保持不变；媒体字节、播放、Range、HEAD、鉴权、数据库和计费不变。
- 验证：视频任务组件定向测试 8/8，任务日志相关两文件 15/15；控制器媒体代理/随机 MP4 Content-Disposition/Range/data URL GET-HEAD 定向测试通过；typecheck、目标 oxlint、oxfmt、git diff --check、前端生产构建、根 Go 构建均通过。
- 本地运行：重建并启动新 Go 二进制，PID 54247 监听 3000；GET /api/status 返回 success=true、版本 v1.0.0-julong.1。未修改用户数据，未生产部署。
- Git：提交 b252d658d（fix: randomize task video download filenames）已成功推送至 origin/main；未推送 upstream、未 force-push。既有 controller/media_asset.go.bak、web/pnpm-lock.yaml、web/src/features/usage-logs/components/media-assets-table.tsx.bak 未跟踪文件排除。

### B09：素材管理 UI、筛选与重新构建（2026-10-15，本地已激活，功能已推送）

- 用户目标：素材管理时侧栏“任务日志”仍选中；管理员独立用户列；时间/用户名筛选；详情弹窗不溢出；重新构建 Go 并启动项目。
- 接续核对：实际 main / f0870f9d1，未处于 merge；origin 已核实为 Xujs98/julong-api。之前代理有 9 个业务/翻译文件改动，尚无本批验证/构建；旧进度快照的功能分支/PID已过时，不据其覆盖代码。既有未跟踪 pnpm-lock.yaml 和 .bak 保留，不纳入交付。
- 正在修正：侧栏 assets 未登记 activeUrls；筛选草稿不应立即请求；旧翻译误写到 translation 外；弹窗需要复用共享 Dialog 并限制媒体高度。现 PID79606 是 09:56 启动的 go run 旧产物。
- 验证、构建、重启、提交/推送均待本轮实际执行，不宣称通过或上线。

### B09本地验证与重建检查点 — 2026-10-15 10:31:46（Asia/Shanghai）

- 完成：素材侧栏归属、独立用户列、时间/用户名搜索重置、列显隐、共享响应式预览；管理员仅本人由后端强制隔离。旧代理的翻译位置错误已撤销，七语言复用原键。
- Go：根go build -ldflags "-X github.com/QuantumNous/new-api/common.Version=v1.0.0-julong.1" -o /tmp/julong-media-ui-rebuild-20261015/new-api . 通过；前端VITE_REACT_APP_VERSION=v1.0.0-julong.1 bun run build通过；旧79538/79606已停止，PID85506监听3000。状态成功、6资源与dist一致、匿名素材401，/tmp/julong-media-ui-rebuild-20261015/activation.json留证。
- 三库：SQLite3.50.4、MySQL5.7.44、PostgreSQL16.15，专用临时库每库7例21/21无Skip；素材service/router定向亦通过。run_matrix.py内部命令和原始日志同目录；没有改schema/驱动/计费，不需迁移变更矩阵。
- 前端：素材4项+侧栏媒体/审计9项定向13/13，6项被过滤Skip；完整两文件另有2个Security & Access无关旧期望失败，明确未通过，不删测试掩盖。typecheck、5文件目标oxlint/oxfmt通过，全量Go未运行。
- 浏览器隔离真实组件夹具验证独立列/搜索/重置/列菜单和600×2400高图长URL预览；对话框768×690且scrollWidth=clientWidth，标题/复制/关闭均在视口。没有真实管理员登录态，不声称已登录页面点测；Safari/生产未测。
- DEVELOPMENT.md同批同步API新参数、页面组件、验证和变更日志。仅提交本批业务/集中回归及文档；既有pnpm-lock及.bak保留未纳入。Git交付待执行，不宣称生产上线。

### B09验证收尾

完整两文件实际重跑：17通过/2失败（Security & Access旧期望），无Skip；素材4例全部通过。定向13通过/6按名称过滤不运行。数据库最终重跑21/21无Skip、版本原始输出已记录，controller/router/service三包通过。最终typecheck、目标lint/format、git diff --check通过；PID85506持续健康。临时浏览器布局夹具完成验证后关闭，不改真实用户或认证配置。

### B09 Git交付 — 2026-10-15 10:34:03（Asia/Shanghai）

- 功能及文档提交8e9a702ce（fix: improve media asset filters and preview layout）已成功推送origin/main，首次push成功；目标仅Xujs98/julong-api，未推upstream、未force。
- 本地PID85506继续监听3000，健康检查通过，最新前端已在Go产物中；浏览器临时4325夹具已停止并关闭，无真实用户数据改动。生产未部署。
- 工作区仅保留本批接续前已有未跟踪controller/media_asset.go.bak、web/pnpm-lock.yaml、media-assets-table.tsx.bak，不纳入提交。测试边界与已知旧失败见上文，不能称全量通过。
- 下一步：在已有登录态的浏览器刷新素材页；生产发布另行操作。


### B11：模型 API 使用开关（2026-10-15，已完成本地验证，提交待执行）

- 用户目标：模型管理→模型→编辑增加模型开关，默认开启，关闭后服务端禁止用户调用。
- 接续核对：main / f58e64561，未处于 merge；现有模型开关初稿尚未通过完整验证；PID54247仍是B10旧产物。仅接续本批代码，保留三个既有未跟踪 .bak/pnpm-lock 文件，不纳入提交。
- 当前工作：默认值/保存/匹配、HTTP及Responses WebSocket请求拦截、普通用户模型过滤和三库新建/升级验证均已完成；模型广场 status 保持独立。
- 验证：前端生产构建、根 Go 构建、定向测试、三库 8/8 矩阵、typecheck、i18n sync、git diff --check 均已实际通过；尚未提交/推送。


### B11 收尾检查点（2026-10-15，验证、重建与本地重启已完成，提交待执行）

- 模型开关已实现：模型编辑页默认开启；后端字段 enabled 兼容旧 NULL，关闭后拒绝普通 HTTP、视频 remix、Responses WebSocket，并过滤普通用户模型列表；模型广场 status 保持独立。
- 实际验证：SQLite 3.50.4、MySQL 5.7.44、PostgreSQL 16.15 fresh/legacy upgrade 与重复 AutoMigrate 矩阵 8/8 无 Skip；controller/relay 定向 Go 测试通过；前端 metadata-editing.test.tsx 8/8、bun run typecheck、bun run i18n:sync 通过。
- 已完成：前端 Rsbuild 生产构建；根 Go 构建输出 /tmp/julong-model-toggle-20261015/new-api；停止旧 PID 54247 并启动新 PID 17798；GET http://localhost:3000/api/status 返回 success: true、版本 v1.0.0-julong.1。三项既有未跟踪 .bak/pnpm-lock 文件不纳入。

### B11 Git 交付检查点（2026-10-15 22:04:25，Asia/Shanghai）

- 本地验证和重启已完成；当前 PID 17798 监听 3000，旧 PID 54247 已停止。
- 模型开关、测试和文档修改已提交为 8a281bc28，并成功推送 origin/main；既有 controller/media_asset.go.bak、web/pnpm-lock.yaml、web/src/features/usage-logs/components/media-assets-table.tsx.bak 未纳入。
- 未进行线上业务查询或生产部署；本地健康检查不等同线上发布。

### B12：安全清理本地构建缓存（2026-10-15，已完成）

- 用户目标：检查并安全清理项目构建产生的缓存碎片。
- 已清理：Go 编译缓存、测试缓存和 fuzz 缓存，执行 go clean -cache -testcache -fuzzcache；检查确认项目内没有 web/.cache、web/.rsbuild-cache、web/node_modules/.cache、.turbo 等前端构建缓存目录。
- 明确保留：当前服务使用的 /tmp/julong-model-toggle-20261015/new-api、web/dist、Go 模块缓存、源码、数据库和既有未跟踪备份文件均未修改；没有停止服务，也没有删除旧验证日志或备份。
- 验证：Go 构建缓存清理后约 8 KB；PID 17798 仍监听 3000；GET /api/status 返回 success=true、版本 v1.0.0-julong.1。本轮未重启、未重新构建、未改用户数据。


### B13：客服群二维码与顶部入口（2026-10-16，进行中）

- 目标：客服配置增加 QQ群/微信群，保留显示名称及联系方式，二维码选择后可预览/删除；搜索右侧、公告左侧显示客服图标。
- 接续核对：main/88b438c7e，origin为Xujs98/julong-api，三个既有未跟踪文件保留。此前摘要为方案，尚无本批业务改动。
- 实现方案：复用客服管理/弹窗、共享 Dialog/CopyButton；二维码和联系方式一起保存（兼容原JSON PUT，新增multipart PUT），避免未保存的上传产生孤立文件；随机本地文件存储，保存失败回滚，成功替换/移除清理旧文件。
- 验证/构建/重启/Git交付待执行；不查询生产、不继续已暂停的其它任务。


### B13 验证与本地激活（2026-10-16）

- 实现：QQ群/微信群 + 二维码选择、预览、删除，保留显示名称/联系方式；顶部Search右侧、NotificationPopover左侧复用紧凑客服按钮。修改通过保存统一生效，失败不清除草稿；旧普通QQ/微信/手机兼容。
- 文件安全/生命周期：5 MB每图、4096像素、总请求16 MB；验证真实PNG/JPEG/WebP和完整解码；随机文件名、原子写入；仅引用既有本站二维码URL，拒绝外部URL/路径穿越；后续验证或数据库失败回滚新文件，成功替换/移除清理旧文件。位置为SITE_ASSET_STORAGE_DIR/support，无数据库表/字段迁移。
- 后端：python3 /tmp/julong-support-20261016/run-matrix.py 内部运行 go test ./controller -run 'Test(SupportContact|SiteLogo|ValidateSiteLogo)' -count=1 -v；SQLite3.50.4、MySQL5.7.44、PostgreSQL16.15 三库群二维码生命周期通过，无Skip。含旧联系方式、两种群、保留字段、读取、替换、删除、非法输入、重复ID、解码截断、文件及数据库失败回滚。原徽标测试有既有audit_logs缺表日志，测试通过不代表审计功能验证。
- 路由：go test ./router ./middleware -run 'Test(AccessTokenRouteRulesCoverEveryDashboardRoute|AccessTokenCatalogCoversEveryPermission|UserAuthAppliesAccessTokenRouteRules)' -count=1 通过。PUT继续沿用UserAuth/AdminAuth/客服配置权限，GET图片公开随机路径，没放开联系方式列表。
- 前端：3文件9/9 Vitest通过，含真实AppHeader入口排序和键盘操作；bun run typecheck、8文件oxlint/oxfmt、bun run i18n:sync、bun run build通过。i18n脚本按规范同步七语言；另三处既有键仅排序归位，没有改历史翻译。
- 构建/重启：根 go build -ldflags "-X github.com/QuantumNous/new-api/common.Version=v1.0.0-julong.1" 成功；旧PID17798正常停止，新PID48103运行 /tmp/julong-support-20261016/new-api，监听3000。健康success=true，首页6资源SHA256与web/dist一致，匿名客服列表401，非法二维码404，activation.json留证。
- 复用审核：共享Dialog、CopyButton、SettingsSection、Input/Select及原客服弹窗；新SupportQRCodeField只是群文件草稿组件。后台数据加载失败禁止保存，避免把错误当空列表覆盖；已登录才显示客服按钮。
- 边界：未取得真实管理员浏览器登录态；组件交互和HTTP已测，真实浏览器端到端/手机扫码/生产未验证。无全量Go测试或生产部署声明。不继续用户已暂停的其它任务。
- DEVELOPMENT.md同步API合同、持久化目录、配置、组件与验证。Git提交/推送待执行；仅本轮文件，原.bak/pnpm-lock不纳入。


### B13 Git交付（2026-10-16）

- 功能/测试/文档提交0e347030e已成功推送origin/main，首次push成功；未推upstream、未force。PID48103本地健康；api.julongkj.top未部署。
- 最终工作区只保留原有未跟踪controller/media_asset.go.bak、web/pnpm-lock.yaml、media-assets-table.tsx.bak；未修改或提交这些文件。此交付记录随后单独同步。

### B14：管理员任务日志权限与 Root 隐私（2026-10-16，进行中）

- 目标：四项独立跨账户日志权限；选中隐私限制后过滤Root日志、禁用用户列表Root详情入口。
- 基线：main/b64990956，无merge；既有.bak、pnpm-lock和site-assets保留，不纳入提交。
- 方案：复用authz权限目录、覆盖策略和PermissionMatrix；统一服务端用户范围用于列表、计数、详情。四项日志权限默认保留已有管理员全站行为，隐私限制默认不选。既有审计Root默认隔离继续保留。
- 验证：尚未运行；需定向回归、真实三库查询矩阵、前后端构建、本地重启和origin/main交付。未生产部署。

### B14 实现与验证里程碑（2026-10-16）

- 四项任务日志独立授权与Root隐私限制已实现，列表/total、任务请求/详情/成果、生图详情/图片、素材删除及Root详情订阅旁路均服务端校验；默认管理员全站、隐私默认false、Root不限。复用现有authz/PermissionMatrix和编辑器；权限保存递增AuthVersion并撤销旧会话。
- 三库真实查询矩阵matrix-v5通过（exit=0）：SQLite3.50.4、MySQL5.7.44、PostgreSQL16.15，独立主/日志库；含五类日志/计数/分页、各类别独立撤权、隐私、直接访问拒绝、文件不被越权删除、用户隔离、Root访问、查询失败拒绝和策略重载。会话更新验证含真实二次凭据、旧会话失效和凭据重放拒绝；后两库另行重复会话用例。无表列迁移。
- 前端权限7/7（permission-ui-v8）、访问/素材/权限3文件15/15（ui-final）通过；typecheck-final、authz全包、routes-final、affected-backend-final通过。临时验证和备份位于/tmp/julong-log-permissions-20261016。最后格式/lint/i18n、详情回归、构建和重启待收尾。
- 既有宽范围回归失败已留证：3组AdminUser测试涉及manage_disable预期与夹具缺表；HEAD版本controller/user.go overlay同样复现，非完整干净基线测试，不宣称全量通过。ClickHouse和真实管理员浏览器端到端未验证。
- 已签发capability成果链接与公开输入素材URL保留原分享语义，不因后台日志隐私开关撤销；不能声称已复制/下载内容被收回。
- 用户询问的任务插件：权限实际是渠道中的绑定；/task-plugins仅Root管理，侧栏当前无入口。本批仅解释并同步文档，没有增加插件页面或功能。
- DEVELOPMENT.md已同步API参数/响应/权限、组件、路由和安全边界。仍需最终构建/本地重启、范围提交及origin/main推送；未生产部署，既有4项未跟踪内容不得纳入。

### B14 最终验证与本地激活（2026-10-16，本地已完成，Git待交付）

- 完成：Root用户更新页复用原权限矩阵显示绘制/生图/任务/素材四项跨账户权限，以及“不允许查看Root账户”。撤销分类权限时只看本人；隐私选中时排除Root五类日志及计数，限制详情、图片、成果和删除入口；Root用户名变为不可点击。后端逐请求校验，权限变更撤销旧会话。
- 最终验证：matrix-v5=0，真实SQLite3.50.4、MySQL5.7.44、PostgreSQL16.15，独立主/日志库，类别隔离/Root隐私/直接访问/文件保留/查询失败拒绝/权限重载/旧会话撤销/二次凭据重放拒绝均通过；无schema迁移。authz-final、routes-final、affected-backend-final均exit0。命令和日志见/tmp/julong-log-permissions-20261016；完整矩阵入口run-matrix-v5.py。
- 前端5文件54/54：ui-final为3文件15项，ui-details-final为2文件39项；permission-ui-v8单独7项属于重复定向检查，不重复累计。typecheck-final、lint-final-v2（18文件）、format-check-v2、i18n-final、frontend-build-final、go-build-final全部exit0；临时翻译脚本已移除。
- Go最新产物/tmp/julong-log-permissions-20261016/new-api；旧PID48103正常停止、新PID24268监听3000，/api/status成功且版本v1.0.0-julong.1，6个首页JS/CSS SHA256与web/dist一致，默认task_logs四项策略已初始化。activation.json留证；没有重复重启或改线上环境。
- 验证边界：全量Go未通过验收；较宽AdminUser回归三组既有失败在基线user.go overlay仍复现（不是完整干净基线），详见上一里程碑；真实管理员浏览器端到端、ClickHouse及全站迁移矩阵未测，不宣称整体ASVS合规。公开输入素材/已签发capability保持既有共享语义。
- DEVELOPMENT.md已同步API/参数/响应/权限、默认值、组件复用、缓存和安全边界。Git提交/推送待本轮执行；仅B14范围，四项既有未跟踪内容保留且不纳入，未生产部署。

### B14 Git交付与最终移交（2026-10-16 18:32，Asia/Shanghai）

- 权限管理功能、集中回归和同步文档已提交为8c9403baee260db750f007913d677d2ea0316b95，并首次成功推送origin/main；git ls-remote确认远端SHA一致。仅向Xujs98/julong-api推送，无force/upstream操作。此交付回执作为独立文档提交随后保存。
- 本地PID24268监听3000且健康，最新前端已嵌入；三库与前端54项等定向验证见上一记录。全量Go既有失败、未测真实登录浏览器和ClickHouse及共享URL边界保留，不扩大通过范围。api.julongkj.top未部署。
- 43个本批文件已范围审核；保护的版权头保持原样。原有controller/media_asset.go.bak、site-assets/、web/pnpm-lock.yaml、web/src/features/usage-logs/components/media-assets-table.tsx.bak保持未跟踪，不包含在提交中。
- 用户最新要求只完成权限管理；本轮没有额外增加任务插件入口。后续在可用真实登录环境补人工验收，生产发布另行执行。

### B15：禁用账户全站拦截与自动追封（2026-10-16，进行中）

- 基线：main/bdba71e34，origin为Xujs98/julong-api；既有4项未跟踪内容保留。
- 用户目标：禁用身份的全站请求拦截、可信身份新IP/设备自动封禁、登录IP显示、SSRF设置页管理入口与登录/注册人机验证核对。
- 已核实：原实现禁用账户并撤销会话、封历史IP；Turnstile已接入登录/注册。全站公共路由和新IP追封尚缺。
- 实现边界：只使用验签凭据/服务端会话秘密识别身份，不凭用户名、原始设备UUID或任意代理头自动封禁；入站黑名单复用BlockedIP，与SSRF出站白名单分离。
- 认证控制：服务端统一检查、禁用会话不可恢复、不可伪造身份触发封禁、Cookie完整性和安全属性、日志无凭据；已读OWASP Authentication/Session Management/Authorization，最新稳定ASVS版本联网核实受限，未编造版本或声称合规。
- 状态：正在实现；三库、回归、构建、重启、提交、推送均待本批验证；未部署线上。

### B15 实现与验证里程碑（2026-10-16，构建收尾中）

- 实现全站SiteAccessGuard、禁用可信身份的新IP/设备追封、正确密码后的追封、签名设备标记（成功登录/刷新写入）、自动IP补充用户详情、SSRF页面独立入站黑名单管理GET/PUT与对应权限/PAT scope和操作审计；出站SSRF配置保持原语义。详见DEVELOPMENT.md接口及B15章节。
- 安全回归涵盖无凭据/伪造用户ID、设备UUID、JWT、refresh和XFF不投毒；跨源禁用凭据拒绝但不持久封禁；正常/已删除账户、重新启用和Cookie过期；真实禁用账户的JWT/PAT/relay/provider/Gemini/refresh/设备换IP追封，IP拦截覆盖官网/静态/API，手动解封恢复。
- 真实三库`python3 /tmp/julong-site-ban-20261016/run-matrix-v2.py`通过exit0：SQLite3.50.4、MySQL5.7.44、PostgreSQL16.15各9顶层用例，无Skip，独立临时主/日志库已删除；首轮MySQL测试库latin1拒绝的失败保留，修utf8mb4后通过。无schema变更。
- 正常密码登录与refresh签发有效签名设备标记、禁用后换IP拦截的追加回归`login-proof`通过；补充三库登录回归运行中。验证码测试模拟Cloudflare成功/失败返回，不能算真实线上验证码验收。
- 前端final-frontend-checks exit0：oxfmt、目标3文件oxlint、bun run typecheck、UI1文件4/4、bun run i18n:sync通过。较宽controller认证回归5例失败，在HEAD业务代码overlay复现相同5例：two_fas夹具缺失、aff_code唯一冲突及已有加密登录失败，证据auth-regressions/controller-baseline.log；model/middleware定向通过、service无选中测试不计通过。
- 本地Turnstile未开启且未配置Site Key；现有入口为系统设置→认证设置→机器人防护，须用户实际域名密钥。未查询生产账户或生产设置。截图请求401/脚本UA/154条不足以认定DDoS或越权。
- 验证局限：浏览器真实管理员交互及线上环境未验；全量Go未通过验收；稳定ASVS联网核实受证书/搜索空返回限制，不声明合规。共享IP可能误封，清Cookie换IP匿名不可归因；CDN缓存/独立官网/流量攻击需要边缘防护。
- 下一步：最终构建、后端重启/健康/嵌入资源核对、范围提交并推origin/main；现仍B14 PID24268运行，未生产部署。四项既有未跟踪内容保留排除。

### B15 最终验证与本地激活（2026-10-16 19:17，本地已完成、Git待交付）

- `router-final`最终10组TestSiteAccess通过exit0，含拒绝响应no-store及实际登录/refresh发标记；`login-matrix`在真实SQLite3.50.4、MySQL5.7.44、PostgreSQL16.15追加登录/refresh/禁用换IP用例均通过。主体三库9组矩阵见matrix-v2；无Skip、测试临时库已清理。测试仅使用临时夹具，未改实际用户封禁或验证码配置。
- `final-frontend-checks`（格式、3文件lint、typecheck、4项UI、i18n）全部通过；frontend-build-final及go-build-final exit0，版本v1.0.0-julong.1；Go嵌入最新dist。控制器5项较宽认证测试在HEAD业务代码overlay复现相同失败，不宣称全量Go通过。
- 本地旧PID24268已正常停止，新PID32714监听3000，/api/status success=true，首页6项JS/CSS SHA256与web/dist一致，新管理接口匿名拒绝。证据/tmp/julong-site-ban-20261016/activation.json。未部署api.julongkj.top，不将本地重启当作线上防御生效。
- DEVELOPMENT.md同步GET/PUT接口参数/响应/权限、禁用及IP统计变化、Cookie语义、组件复用、SSRF出入站分离、共享IP/匿名识别/CDN缓存边界和验证码开启方式；稳定ASVS联网核实与真实浏览器/线上验证码验收仍有限制。
- Git交付待执行：仅B15范围，保留并排除controller/media_asset.go.bak、site-assets/、web/pnpm-lock.yaml、web/src/features/usage-logs/components/media-assets-table.tsx.bak；目的地仅origin/main，禁止force/upstream。

### B15 Git交付与最终移交（2026-10-16 19:22，Asia/Shanghai）

- 功能与同步文档提交8a672a54fe1cf361967d799e8b3f1c7c0a79e146，首次推送origin/main成功，git ls-remote核对远端main SHA一致。本段和DEVELOPMENT.md交付回执作为独立文档提交随后保存并推送；不使用upstream或force。
- 本地PID32714监听3000，健康/最新嵌入资源/匿名管理拒绝已验证；最终10组、三库和前端4项等证据见上。已知controller 5项基线overlay失败、真实浏览器/验证码/ASVS联网核实局限保留，不宣称全站无风险或全量测试通过。api.julongkj.top未部署。
- 26个B15文件范围审核并提交；四项历史未跟踪文件/目录保持未跟踪，未清理缓存或用户数据。本批没有查询线上目标用户、写线上配置或启用未提供密钥的验证码。
- 后续生产发布需按实际部署流程上线，再配置真实Turnstile域名密钥、可信代理和CDN/WAF边缘策略；共享IP误封可解封，重新启用账户不会自动解封已有IP/设备。

### B16：使用日志权限与用户在线统计（2026-10-16，进行中）

- 目标：新增使用日志跨账户权限并继承Root隐私；概览日活/在线排除Root；用户名和详情在线标记，状态筛选在线/离线，Root状态对所有人隐藏。
- 基线：main aa4aba641；四项既有未跟踪内容保留；无未完成merge。
- 实施：复用UserPresence与权限目录、UserLogScope、共享表格筛选/状态组件；当前记录仅监控用户，需使活动记录独立于邮件开关。
- 验证/构建/重启/提交/推送：尚未执行；未部署生产。


### B16 实现与验证里程碑（2026-10-16）

- 使用日志独立权限、Root隐私覆盖列表/计数/统计/CSV/NDJSON，前端缓存按权限隔离；概览日活/在线卡片、用户列表/详情在线角标和状态筛选已实现。Root不采集新活动、不返回状态、不进入任一在线筛选/统计。
- 在线最近300秒，日活UTC+8当天用户去重；复用UserPresence无schema新增，记录覆盖所有已认证启用非Root账户，邮件独立受原配置控制；前台可见60秒心跳、展示30秒刷新。
- matrix-v1 exit0：真实SQLite3.50.4/MySQL5.7.44/PostgreSQL16.15独立主/日志库；日志权限/隐私/分页/导出/额度RPMTPM、Root提升前记录、在线边界/跨日/禁用/删除/从未活动、HTTP列表/搜索/详情隐藏、权限修改撤销旧会话/一次性凭据重放拒绝通过，无Skip。临时库已清理。测试夹具附加用户元数据缺表日志不代表该类功能已验证；故意缺users表验证fail closed输出错误为预期。
- backend-presence exit0：service5组、router2组实际运行通过；authz/middleware无匹配仅编译。七语言六键已脚本同步并删除脚本。前端格式/lint/typecheck/UI收尾中，未构建/重启/提交/推送。
- DEVELOPMENT.md已同步接口/权限/参数/响应/组件/活动采集和状态边界。历史全量Go失败仍保留，真实管理员浏览器/ClickHouse/生产未验证，不宣称整体ASVS合规。四项既有未跟踪内容保留排除。


### B16 最终验证与本地激活（2026-10-16 19:52，本地完成、Git待交付）

- 完成使用日志独立跨账户授权、Root隐私覆盖列表/计数/统计/CSV/NDJSON；概览当日日活/在线、用户列表/详情绿色在线/灰色离线角标、原状态筛选在线/离线。Root状态对所有查看者省略，两个筛选和两项人数均排除Root。
- 真实SQLite3.50.4/MySQL5.7.44/PostgreSQL16.15矩阵matrix-v1、后端定向backend-presence和authz全包authz-final均exit0，具体命令/范围见上一记录；无新增schema。权限保存、旧会话撤销和凭据重放拒绝已有实际回归。
- ui-final-v2 exit0：3文件17/17（使用日志权限/缓存、管理员保存、统计卡/在线徽标/筛选）；16文件oxlint/oxfmt、bun run typecheck通过。i18n-final、frontend-build-final和go-build-final均exit0；gofmt -l和git diff --check干净。没有运行全量Go，不掩盖历史失败。
- 在线按最近300秒有效认证活动；日活按UTC+8当日去重。原邮件配置独立；可见登录页面60秒心跳、展示30秒刷新，关闭/退出后最多5分钟离线，非物理连接计数；部署前未采集的日活不回溯。
- 已使用SQLite backup API安全备份并quick_check通过（临时目录文件权限0600），正常停止旧PID32714，新PID38584运行/tmp/julong-presence-20261016/new-api并监听3000。健康success=true，版本v1.0.0-julong.1，6项首页资源与dist SHA256一致；两个新增活动接口匿名401。activation.json留证；未重复重启、未改生产环境。
- 本轮收尾已成功读取OWASP官方ASVS页面（稳定版本5.0.0）及Authorization、Authentication、Session Management Cheat Sheet；先前网络失败记录保留。按服务端逐请求授权、失败拒绝、权限变更撤销旧会话及不返回Root状态核对；未做整体ASVS认证，真实管理员浏览器、ClickHouse、生产未验证。
- DEVELOPMENT.md已同步API合同、权限/参数/响应、组件、采集口径和验证。Git提交/推送待执行；四项既有未跟踪内容排除保留，不继续历史取消任务。


### B16 Git交付与最终移交（2026-10-16 19:56，Asia/Shanghai）

- 功能/测试/同步文档提交043f1adc862816a545a9c9339cb2b9187abe3f6a已首次成功推送origin/main，git ls-remote确认远端SHA一致。38个本批文件已范围审核；未推upstream或force。此交付回执由独立文档提交保存。
- 本地PID38584监听3000，健康与6项嵌入资源核对通过；真实三库、前端17项与定向权限/活动测试、格式/typecheck/i18n/build结果见上一记录。Root不采集/不统计/不返回在线字段，线上未部署。
- 四项既有未跟踪内容controller/media_asset.go.bak、site-assets/、web/pnpm-lock.yaml、web/src/features/usage-logs/components/media-assets-table.tsx.bak保持未跟踪，未删除或纳入提交。全量Go、真实管理员浏览器及ClickHouse未验收；不扩大验证结论。
- OWASP官方页面工具本轮返回空后，已通过正常TLS的curl读取并保存ASVS/Authorization/Authentication/Session Management原文于临时验证目录，稳定版5.0.0已从官方正文核实；未关闭TLS校验，未声称整体合规。
