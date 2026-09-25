# 进度

## 2026-09-24

- 用户确认单一原生输入已在真实 iPhone Safari 可用，阶段 21 完成；`/model` 已能打开，用户确认可用手机键盘操作原生选择器，决定暂不增加触摸选择层。

- 用户新截图指出 tmux copy mode 回滚光标脱离 Codex 原生输入栏，且页面有两个输入位置；用户确认原生栏可执行指令，要求移除网页 composer。隔离 tmux 与浏览器 DOM 定位：`[1/127]` 表示 copy mode，白框是其定位光标；失焦时隐藏 outline，聚焦时只在 `pane_in_mode=1` 时退出回滚。隔离 tmux 还发现必须精确用 `=session:0.0` 和 `send-keys -X -t ... cancel`，已据此修正服务端测试与实现。
- 已移除重复 composer、无用附件占位与页内“已发送”列表；本机浏览器探针从 23 行增至 28 行，原生输入聚焦、触摸滑动释放焦点和模拟键盘可见高度通过。服务端 WebSocket/Manager 定向测试通过；待全量检查、构建和真机刷新验证。
- 补充浏览器触控时序 RED/GREEN：`touchstart` 后 xterm 先聚焦、随后 `touchmove` 的滑动原本会提前发送 `terminal.focus`；现在仅轻点结束后发送聚焦，真正滑动保持 tmux 历史模式。浏览器轻点和滑动探针分别通过。
- `make check`、前端语法和差异检查通过，构建并重启私有 Serve 后端；线上 `/healthz`、页面和资源均返回 200，页面已无重复输入框，四个原有 Codex pane 存活。已请求用户刷新 iPhone Safari 验证回滚光标及原生输入。

- Ghostty 风格移动端终端已完成本机实现：内嵌 JetBrains Mono Regular/Bold 与 OFL，采用本机 Ghostty 默认深灰 ANSI 色板；移动端压缩周边控件，让原生 TUI 占用剩余高度。首次预览发现字体晚于 xterm 初始化导致字符过疏，改为先等待字体加载后正常。窄屏探针从 17 行提升到 23 行、61 列，截图检查通过；`make check`、JavaScript 语法和连续两轮提交探针通过。待部署与真实 iPhone Safari 复核。
- `make build` 后重启当前 launchd daemon；localhost 与私有 Serve `/healthz` 返回 200，JS/CSS 与两份 WOFF2 均已由新进程服务。当前受控终端列表仍可读取；已请求用户在真实 iPhone Safari 刷新确认字体和键盘布局。
- 用户真机截图指出终端正文仍没有颜色或粗体。以同一 Codex pane 的 tmux ANSI 控制码、浏览器 xterm class 和计算样式逐层定位到 CSP：`style-src 'self'` 阻止 xterm 的运行时页内样式表。浏览器级探针 RED 时颜色全白/字重 400，修复后 GREEN 显示绿色代码路径/字重 700；`make check`、构建和私有 Serve 响应检查通过，已重启 daemon 并请求 iPhone 刷新确认。
- 用户在真实 iPhone Safari 刷新后确认终端视觉“已经非常好”；阶段 20 视觉工作完成。键盘弹起/横屏等通用交互项继续在原验收表跟踪。

- P0 插队：用户确认中文显示恢复，但两个新 Codex 对话都只回应首轮“你好”。两个真实 tmux pane 的第二轮中文任务停在原生输入框；对其中一项单独发送 Enter 后 Codex 立即开始执行，证明文字已到达但同批回车未提交。
- 浏览器点击探针 RED：连续两次点击“运行”分别发出短问候和较长中文任务，两次均缺少 paste 边界；GREEN：普通 Codex composer 复用已验证的结束边界，两次点击均捕获到正确序列。`make check`、JS 语法、构建和私有 Serve 静态资源检查通过；新 daemon 已重启，四个原有 Codex pane 保留。待 iPhone Safari 刷新后真机复验。
- 原先 Ghostty 风格视觉任务按用户要求让位于 P0；已记录用户同时希望改善终端字体/配色与可见高度。本机 Ghostty 为默认配置，Chrome 窄屏基线终端为 17 行；视觉代码尚未改动，待 P0 真机验收后继续。
- 用户确认新版 iPhone Safari 在同一 Codex 对话里连续两轮均正常回复，阶段 19 完成，恢复 Ghostty 风格视觉任务。
- 用户截图显示 Codex 原生 TUI 中中文变成连续横线；用户确认退出 tmux copy mode 后仍如此。tmux `capture-pane` 保留正确中文，排除 Codex 生成内容缺失；运行中的 launchd daemon 没有 UTF-8 locale。
- 独立 tmux PTY 探针在无 locale 时复现 `Codex/Claude ______ TUI`，加 tmux 全局 `-u` 后收到原始中文 UTF-8。先将 `-u` 写入附着命令回归断言并观察失败，再实施单一修复；`make check` 通过。
- `make build` 后已重启现有临时 launchd job；新 daemon 运行中，localhost 与 tailnet-only Serve `/healthz` 均返回 200，原有两个 Codex tmux pane 仍在。等待真实 iPhone Safari 刷新确认中文。

## 2026-09-22

- 用户要求先恢复服务并清理现有 11 个终端，再实现手机端终端清理与电脑原生 session 恢复。已逐个核对并结束 `code-remote` 专用 tmux server 中的 11 个受控 session；未触碰普通 tmux。
- 清理脚本在最后一个 tmux session 结束、server 正常退出时提前停止，未执行同一脚本中的最终 daemon 重启；随后单独确认旧 daemon 和 tmux server 均已退出，再完成重启。localhost API 与 Tailscale Serve `/healthz` 均通过，终端 API 只保留新的逻辑 `prototype` 入口。
- 阶段 17 启动：按 TDD 实现显式 DELETE 生命周期和手机确认交互；Codex 原生历史继续作为恢复事实源，不把终端删除扩展成 provider 历史删除。
- 当前工作区基线 `go mod download` 与 `make check` 全部通过；现有未提交的 Codex bracketed-paste 修复保持不动，可以进入阶段 17 的 RED 测试。
- Catalog 删除合同进入 RED：新测试要求只结束指定受控 session、保留其他 session，并让重复删除幂等；当前因 `Catalog.Delete` 尚不存在而按预期编译失败。
- Catalog GREEN：`Delete` 只对 catalog 已知 ID 发出参数数组形式的精确 `tmux kill-session`，成功后才移除内存条目；重复删除不再次触碰 tmux。定向测试通过。
- HTTP 删除合同进入 RED：同源 `DELETE /api/v1/terminals/{id}` 期望 204，跨源请求不得到达 catalog；现有动态路由只允许 GET attach，定向测试按预期得到 405。
- HTTP GREEN：`SessionCatalog` 暴露显式 Delete，动态终端路由区分 GET attach 与 DELETE terminal；同源删除返回 204，跨源删除返回 403，原有未知 attach 行为保持。定向测试通过。
- 手机交互进入 RED：测试要求每个运行终端具有明确结束按钮、原生确认、DELETE 请求、列表移除和清空后的无活动终端状态，并明确不会删除 Codex 历史；现有页面全部缺失，定向测试按预期失败。
- 手机交互首次 GREEN 后复核发现失败提示会被 `finally` 的无条件列表重绘立即清除。根因是错误仅作为临时 DOM 节点、没有进入渲染状态；补充失败测试后用 `sessionListError` 作为单一状态源，定向测试与 JavaScript 语法检查重新通过。
- 无终端状态复核又发现 `updateRuntimeContext` 会把“暂无终端”覆盖为“新对话/prototype”。先补失败断言，再让会话标题和状态面板显式依据空 `terminal_id` 显示“暂无终端”；定向测试通过。
- 组合回归通过：同源删除运行终端后，Codex 原生 history 仍可按 cwd 列出，选择后继续走 `codex resume` 创建新终端。另用独立临时 tmux socket 实测 `kill-session -t =session` 精确目标有效，未触碰当前 `code-remote` server。
- 全量 `make check`、JavaScript 语法与 diff whitespace 检查通过，新 daemon 已构建。工具 shell 下的 `nohup` 子进程会被宿主回收，因此本次手动部署改用当前 macOS 登录会话内的临时 `launchctl` job `dev.coderemote.manual`；它不是开机自启配置。localhost 与 Serve 健康检查通过。
- 真实 Serve API 探针在 paper cwd 创建 `session-b7864b0aa5b5`，读取到 15 条 Codex 原生历史，并恢复一条到 `session-5633e4d106ce`。两个测试终端均通过 DELETE 返回 204；运行列表只剩 `prototype`，删除后 paper 历史仍为 15 条。测试终端已清理，无测试 agent 残留。
- 最终复验通过：`make check`、`go test -race ./internal/terminal ./internal/web`、JavaScript 语法、diff whitespace、localhost/Serve 健康检查及线上静态资源检查均正常。新版本由临时 launchd job 保持运行，等待用户在真实 iPhone Safari 刷新后验证“结束”和 paper 历史“恢复”按钮。

## 2026-09-21

- 用户报告真实 iPhone 上 `/model` 交互失败，要求完成交互验收。阶段 16 启动；先撤回“实现即支持”的判断，按 Safari 点击 → 前端发送 → WebSocket → daemon → tmux → Codex TUI 的边界逐层取证。
- 初始环境检查发现 Serve 配置存在但 8080 无 daemon 监听，tmux 中 11 个终端均未附着。先恢复当前构建的 localhost daemon，随后再做可观测的端到端命令探针。
- 已完成第一轮数据流检查：按钮调用 `sendNativeCommand`，经 attachment ID 封装为 `terminal.input`，daemon 解码后写 PTY。当前测试覆盖传输层但未覆盖真实按钮事件；另外发送前置条件失败会静默无反馈，是验收可观测性缺口。
- 当前构建已在 localhost 启动，390×844 的真实 Chrome 页面成功建立到 `prototype` 的 tmux 附着（61×17）；现在进入按钮点击复现。
- 完整点击复现已定位根因：`sendNativeCommand("/model")` 的 `/model\r` 同批输入只把文本放进 Codex composer；同一页面再发送独立 Enter 后原生模型选择器立刻出现。下一步以该复现补测试并把“文本输入”和“提交键”拆成可验证的顺序。
- 最小延时实验：文本与 Enter 拆帧但 0 ms 仍失败，50 ms 回车可触发 Codex 提交。正在验证显式 bracketed-paste 结束标记能否提供无需猜测时长的输入边界。
- 全新 Codex 终端上的 bracketed-paste 探针通过：`ESC[200~/modelESC[201~CR` 单帧即可打开原生模型选择器。确定采用显式输入边界修复 Codex 快捷命令，进入 RED 测试。
- RED 已确认：新增回归合同要求原生快捷命令用 bracketed-paste 边界编码；现有实现因缺少编码函数及调用而按预期失败，证明测试能捕获本次缺陷。
- GREEN 已确认：`nativeCommandSequence` 现在生成 bracketed-paste 文本边界，并在需要提交时追加 CR；定向 Go 测试与 JavaScript 语法检查通过。修复后的 daemon 已重新构建并监听 localhost，准备浏览器点击复验。
- 修复后浏览器点击复验通过 `/model` 与原生 `/` 菜单：390×844 页面、WebSocket、PTY、tmux、Codex TUI 全链路均有对应证据。一次串联的 `/status` 用例因残留 `/` 变成普通任务，已立即中断；将从空 prompt 重新独立验证。
- 空 prompt 下独立点击 `/status` 已通过，Codex 显示真实原生状态卡片。浏览器层的三类交互复验完成，下一步跑全量回归并同步协议/验证文档；真实 iPhone 仍需刷新后复验，不能用 Chrome 结果冒充 Safari 结果。
- 全量 `make check`、JavaScript 语法、diff whitespace 检查通过；Serve HTTPS 已返回修复后的静态资源。架构、决策、运行时、协议与验收记录已同步。iPhone peer 在线且修复后的 daemon 正在运行，阶段 16 只剩用户在 Safari 刷新后的实际点击确认。
- 已结束并删除本轮专门创建的测试 tmux 终端 `session-056afca023d4`，避免污染用户终端列表；该测试终端本身不可恢复，但 Codex 原生历史仍由 provider 管理。headless Chrome 进程也已停止，修复后的 daemon 保持运行供 iPhone 复验。

## 2026-09-20

- 用户反馈路径面板逐层到达 Yuniverse 后关闭面板，顶栏仍显示 code-remote。实时 API 证明 Yuniverse 终端 `session-52477ab1b578` 已存在；源码确认目录行点击只调用 browse，真正的 create/switch 按钮位于可滚动列表之后。阶段 15 启动，修复确认动作在 iPhone 上不易发现的问题。
- 阶段 15 完成：先用失败测试复现确认按钮位于滚动列表之后，再把它移到列表上方并根据浏览/输入路径显示“新建并切换到 Yuniverse”。全量 `make check`、构建、JavaScript 语法、重新部署后的健康检查和真实 Yuniverse 目录 API 均通过。
- 用户要求实现并推送工作目录切换，当前目标为 `~/Developer/personal/projects/Yuniverse`；已确认目录存在并解析为 `/Users/huangxinxinyu/Developer/personal/projects/Yuniverse`。阶段 13 启动，将按“新目录新终端、旧终端继续运行”的既有生命周期合同测试驱动实现。
- 用户把后续优先级扩展到 Codex 原生 `/` 指令，模型切换最重要。官方文档已核实交互式 CLI 的 `/model` 合同；阶段 14 将复用原生 TUI 选择器，不在 Web 层维护模型清单或修改用户配置。
- 阶段 13 自动化实现完成：目录 API 支持规范化、home shorthand、直接子目录浏览与无效路径拒绝；terminal create 可提交 cwd，tmux session 写入 cwd 元数据供 daemon 重启恢复；手机路径面板支持手输/逐层浏览并在目标目录新建后切换。Codex 历史读取和恢复同步跟随当前终端 cwd。
- 阶段 14 自动化实现完成：模型面板用 `/model` 打开 Codex 原生模型/推理强度选择器，支持输入 `/` 打开全部原生命令，并提供 `/status`、`/permissions`、`/review` 快捷入口；非 Codex agent 禁用。全量 `go test ./...` 与 JavaScript 语法检查通过。
- 真实 Yuniverse 创建探针首次暴露 tmux session target 差异：session 已启动但 `set-option -t =session-id` 失败并返回 HTTP 500。补充 fake runner 回归后改用有效 target，精确结束该次空白测试残留，再由修复后的 API 成功创建 `session-52477ab1b578`。Daemon 重启后仍恢复 Yuniverse cwd，旧 6 个 session 全部存活，Serve 健康检查通过，Yuniverse 原生 Codex 历史返回 11 条。
- 在新 Yuniverse 空白终端真实触发 `/model`，Codex 0.155.1 成功显示 “Select Model and Effort” 原生列表；未确认模型变更，随后 Esc 并清空输入回到普通提示符。手机端按钮点击和布局仍留给真实 iPhone 最终验收。
- 最终差异审阅发现架构目录树同时保留了未来 `internal/directories/` 与已实现 `internal/directory/`，已删除被取代的复数占位，避免文档给出两个目录服务位置。
- 阶段 13 与 14 主提交 `52298c6`（`Add mobile workspace session controls`）已推送到 `origin/main`；路径切换和原生指令范围完成，剩余真机点击/布局确认继续归入既有 iPhone 验收。
- 用户要求 Codex 黑色原生工作区支持上下滑动。已确认全屏 alternate screen 下仅设置 CSS overflow 不够；采用 tmux mouse + `WheelUpPane` copy mode，并将 TUI 区域单指垂直手势编码为原生鼠标滚轮事件。自动化回归通过，当前专用 tmux server 已开启；真机方向与退出 copy mode 待用户刷新验证。
- 用户明确要求能选择过去对话并使用 Codex 自己的命名。已按官方 app-server 合同实现 `thread/list` cwd 过滤、历史区与 `codex resume <id>` 新终端恢复；线上真实接口返回 6 条，名称包括“确定下一步”“编写 mac 软件 README”“问候用户”。不解析 provider 文件、不建立聊天数据库。
- 阶段 11 完成：composer `+` 只承载图片/文件上下文入口，底栏原“切换项目”已替换为直接“新对话”，project/path 合并为工作目录入口。
- 用户用真实 iPhone 截图澄清入口语义：composer `+` 应留给图片/文件等上下文；“新对话”应替换底栏“切换项目”，因为 project/path 在当前产品中都是 cwd 选择。阶段 11 启动，按测试驱动修正。
- 用户指出 Codex 应已有对话命名。已核实 OpenAI 官方文档的保存/恢复能力、本机 Codex 0.155.1 app-server schema 与真实 `thread/list`：Codex 确实提供原生 `name`；进一步确认 tmux pane title 能无歧义地拿到当前 TUI 的自动标题，阶段 10 启动，准备以测试驱动接入 terminal list API 和手机会话列表。
- 阶段 10 完成：三轮失败测试分别锁定 tmux 标题刷新、terminal API `title` 字段和前端原生标题优先级；实现后全量 `make check`、JavaScript 语法与 diff 检查通过。重启 daemon 后本地与 Serve API 均返回 `问候用户 | code-remote`，390×844 渲染实际显示“问候用户”和尚未命名的“对话 2”。
- 用户将下一优先级设为手机端新建对话。阶段 9 已启动：目标是保留当前终端，同时提供独立 terminal session 的创建、列表、动态附着和切换入口。
- 阶段 9 后端合同已进入 GREEN：terminal catalog 能从专用 tmux server 恢复 `prototype` 与 `session-<12hex>`，创建新 agent 时沿用 cwd/agent 并移除 `NO_COLOR`；Web API 已支持终端列表、同源创建和按 ID 动态附着，未知终端返回 404。
- 阶段 9 手机 UI 已进入 GREEN：底部“终端”和 composer `+` 可进入 session sheet，显示当前项、独立 terminal ID/上下文、新建按钮；创建成功自动切换，WebSocket 使用动态 terminal ID，旧连接的迟到事件由 generation 隔离。全量 Go test/vet、前端语法与 diff 检查通过。
- 阶段 9 真实 Serve 验收创建了第二个 Codex session；旧 `prototype` 与新 session 同时存活，新 session 恢复 ANSI 色彩环境。Daemon 重启后两项均从 tmux 恢复，指定新 terminal ID 的 WSS attach 通过，断开后两个 agent 均继续运行。
- 阶段 9 完成：浏览器会在 localStorage 只记住最近选择的 `terminal_id`；正式产品、架构、决策、运行时、协议、开发与验收文档已同步。真机交互复核继续归入阶段 8 剩余验收。

- 用户已完成 Tailscale 安装准备并要求开始原型开发；Mac CLI 已确认登录，地址为 tailnet 内私有地址，`make doctor` 全部通过。
- 已启动阶段 8：先建立 Git 基线与隔离工作区，再按静态终端交互、单终端链路、跨网双路径验证推进。
- 已创建基线提交 `d6bfec1`；用户明确不使用隔离 worktree，后续直接在当前 `main` 工作区开发。
- 监听边界与 WebSocket 同源校验完成首轮 TDD：仅 loopback/Tailscale 数字地址可监听，缺失或外站 Origin 被拒绝；相关 Go 测试通过。
- 原型浏览器终端锁定 xterm.js 6.0.0 与 fit addon 0.11.0，采用仓库固定 ESM 发布文件并内嵌到 Go 二进制的方式，不依赖 CDN。
- 已实现单终端 Go daemon 原型：xterm.js 手机页面、同源 WebSocket v1、base64 原始终端字节、输入与 resize、PTY 附着、专用 tmux session 复用及单写附着替换。
- `make check`、构建、JavaScript 语法检查和隔离 tmux 冒烟通过；daemon 已只绑定 `100.78.102.15:8080`，Mac 本机通过该 tailnet 地址读取健康检查与页面。
- 当前等待真实 iPhone 打开直连地址，验证 Tailscale peer、Safari 显示与输入；尚未声称端到端跑通。
- 真实 iPhone 已通过 tailnet 直连 `100.78.102.15:8080`：Mac 观察到来自 iOS peer 的已建立连接、一个活动 tmux attach，以及存活的 Codex pane（44×34）。页面、WebSocket、PTY、tmux 与真实 agent 链路已建立；输入细节、断开恢复和长连接仍待用户确认。
- 用户确认真实 iPhone 终端正在工作。切换 Serve 前停止直连 daemon 后，专用 tmux session 的附着数降为 0，而 Codex pane 继续存活，证明 daemon/WebSocket 断开未结束 agent。
- localhost 后端已在 `127.0.0.1:8080` 运行并通过健康检查；Tailscale 1.102.4 返回“Serve 尚未在 tailnet 启用”，需要用户通过 Tailscale 管理链接完成一次性授权后才能继续 HTTPS/WSS 测试。Funnel 未启用。
- 用户已在 tailnet 管理页启用 Serve。当前私有 HTTPS 入口代理到 `127.0.0.1:8080`，Serve 状态标记为 tailnet only，Funnel 未启用；Mac 端 HTTPS 健康检查及真实 WSS attach 探针通过，Codex pane 在探针断开后继续存活。
- 用户将优先级切换为 UI 视觉优化。已使用 390×844 headless 浏览器截图查看当前真实页面，确认需要加强连接状态、终端框架、会话信息和底部快捷键的层级与触感。
- 已完成移动端终端外壳第一轮视觉优化：新增品牌标记、tailnet 副标题、连接状态胶囊与信号轨、终端会话/尺寸栏、石墨终端框架及分组 keycap 工具栏；保持全屏 TUI 为视觉主体。
- 新 UI 的结构回归测试、全部 Go test/vet、JavaScript 语法和 Serve 页面资源检查通过。500px 完整布局截图已人工复核；Chrome headless 最小布局视口为 500px，因此 390px 截图会裁切右侧，真实 iPhone 仍需刷新确认。
- 用户提供成熟控制台参考图并要求整体重构。已完成第二轮 UI：新增真实 runtime context API、Project/Path/Agent 上下文条、可扫描的原生 TUI 运行卡片、独立多行 command composer、字符计数与发送状态、连接详情/切换说明 bottom sheet、终端控制键折叠区和四项 action bar。
- 新版 UI 保持原生 TUI 为权威输出，不解析终端文字伪造聊天、tool execution 或 completed 状态；未实现的 project/path/model 切换在面板中明确说明。500×1000 headless 移动布局与延时 TUI 重绘已人工复核，真实 iPhone Safari 刷新验收待用户执行。
- 根据真机反馈移除冗余终端控制键条。排查确认 agent 输出链路正常，但 composer 发送后键盘未收起会把 pane 压到 45×14；已改为发送后释放焦点并滚回实时终端，恢复可视运行空间。
- 排查移动端终端样式单色问题，确认现有 Codex 进程继承了 `NO_COLOR=1`，上游未产生颜色 ANSI；已让未来新 agent 只移除该变量，并显式加强 xterm normal/bold 字重。当前 tmux/Codex session 未被破坏性重建。
- 用户确认首版采用 Web + Tailscale：iPhone Safari 通过私有 tailnet 连接 Mac 上的 Go Web daemon。
- 已同步 README、产品范围、架构、决策、运行时、协议、开发、验收与接续文档。
- 已明确 Tailscale 只负责私网连通；目录、终端、tmux 生命周期和恢复由本项目负责。
- 已记录两条待验证接入路径：直接 tailnet HTTP/WS，以及 localhost 经 Tailscale Serve 的 HTTPS/WSS。
- 已发现 Serve 在 Safari/HTTP2 与长连接方面存在公开问题，因此不能在真实设备测试前宣称已跑通。
- 默认 bootstrap 只准备 Go 与 tmux；doctor 改为检查 Tailscale和本机 agent。
- 通过测试驱动方式保证默认开发流程只使用当前声明的工具、仓库不含被移除的手机客户端资产，且构建输出位于 `BUILD_DIR`。
- 用户明确要求彻底移除旧手机客户端工程、相关构建入口、忽略规则、文档记录及机器上的旧工程生成器；清理已执行。
- 旧客户端文件此前均未提交，删除后不能从 Git 恢复。机器上的旧工程生成器已卸载；macOS 通用开发命令行工具未动。
- 当前下一步是完成最终验证，然后安装 Tailscale 并开始真实 iPhone Safari 原型。
- 最终验证通过：仓库清理回归测试、Go test/vet/build、Markdown 本地链接与格式、Git diff 检查均正常；Homebrew 与 PATH 均确认旧工程生成器已移除。doctor 唯一待满足项仍是安装 Tailscale。

## 2026-09-19

- 明确第一版只需连接、运行、显示和输入，不做多 agent 协同。
- 明确 project/workspace 只是寻找工作目录，不建立项目注册或绑定业务。
- 明确首版必须跨网，手机中断后 Mac 上的任务继续，重连后恢复终端现场。
- 明确同一目录可以运行多个独立终端，切换页面不能结束 agent。
- 核查 tmux 的 detach/reattach 能力，确定它是终端保活候选；尚未完成端到端验证。
- 核查 Multica 的 CLI 发现机制，采用显式路径、PATH、按需 shell 回退的职责划分，不引入其业务模型。
- 建立 Go module、daemon/relay 占位入口、协议 envelope、环境检查和基础文档。
- 安装 tmux，确认 Go、Codex 与 Claude 在当前 Mac 可发现。
