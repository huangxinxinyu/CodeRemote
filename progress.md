# 进度

## 2026-09-20

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
