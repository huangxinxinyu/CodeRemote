# 产品访谈与研究结论

正式产品范围见 [docs/product.md](docs/product.md)，技术基线见 [architecture.md](architecture.md)。本文件只保留仍适用于当前 Web + Tailscale 方案的结论。

## 用户确认

- 第一版聚焦连接和运行：手机能选择电脑工作目录与 agent，进入原生 TUI 并输入任务。
- 不建立产品自己的聊天数据库；会话历史和 resume 能力交给 agent。
- project/workspace 只是工作目录入口，不建立项目实体、注册或绑定流程。
- 首版必须跨网，不接受只能同一 Wi-Fi 使用的产品。
- 同一目录支持多个独立终端，但不做自动分工、通信、调度或冲突解决。
- 手机锁屏、页面关闭或断网时，Mac 上的 agent 继续运行；回来重新附着。
- 客户端采用 iPhone Safari，网络采用 Tailscale 私有 tailnet。

## Tailscale 接入核查

- 2026-09-20 本机实测：Tailscale CLI 已安装在 `/usr/local/bin/tailscale`，Mac 已登录并获得 tailnet 私有地址；手机 peer 是否在线仍需单独确认。
- Tailscale Serve 可把 Mac 上监听 `127.0.0.1` 的 HTTP 服务通过 tailnet 内的 HTTPS URL 提供给 iPhone，应用 tailnet ACL，并向后端添加身份请求头。来源：https://tailscale.com/docs/features/tailscale-serve
- Serve 需要 MagicDNS 与 tailnet HTTPS；证书域名会进入公开 Certificate Transparency 日志，但服务内容仍仅在 tailnet 内可达。来源：https://tailscale.com/docs/how-to/set-up-https-certificates
- macOS 客户端的 Serve 可以代理本地端口；沙箱限制主要影响直接共享文件或目录，不影响端口代理。来源：https://tailscale.com/docs/reference/examples/serve
- 2026 年仍有公开、未关闭的 Serve WebSocket 问题：Safari/HTTP2 握手失败、连接偶发断开、升级请求查询参数丢失。来源：https://github.com/tailscale/tailscale/issues/20882、https://github.com/tailscale/tailscale/issues/18827、https://github.com/tailscale/tailscale/issues/18651
- 首个原型必须分别验证直接 Tailscale 地址的 HTTP/WS，以及 Serve 的 HTTPS/WSS；若 Serve 不稳定，先保留直接 tailnet 路径。
- 服务不能监听全部网卡或普通 LAN 后仅靠地址难猜保护。直接模式只绑定 Tailscale 地址；Serve 模式只绑定 localhost。
- HTTP API 保持同源，WebSocket 校验 Origin，避免其他网页借用已入网的浏览器调用终端 API。

## CLI 发现

- Daemon 复用 Mac 当前用户已安装并认证的 Codex 与 Claude，不复制 provider 凭据到手机。
- 发现顺序建议为：显式路径、daemon PATH、带超时和缓存的用户 shell 回退。
- 显式路径无效时应报告配置错误，不静默切换到另一份安装。
- 版本探测与可执行文件发现分离；版本命令失败不能伪装成未安装。
- 参考：https://github.com/multica-ai/multica/blob/main/server/internal/daemon/agents_probe.go

## 终端生命周期

- tmux 支持 detach/reattach 和后台 server，适合让 agent 生命周期独立于浏览器连接。来源：https://github.com/tmux/tmux/wiki/Getting-Started
- 每个 `terminal_id` 对应独立 tmux session；浏览器每次连接获得新的 `attachment_id`。
- 页面离开、网络中断和 Tailscale 重连只替换附着，不结束 tmux session。
- 当前屏幕通过重新附着和重绘恢复，不能把历史纯文本当成完整 TUI 状态。
- 首版每个终端只允许一个可写附着；输入在断线后不自动重放。

## 尚未验证

- iPhone Safari 中文输入、特殊键、粘贴、触摸滚动、视口变化和复杂 TUI 重绘。
- Tailscale Serve 的 WebSocket 握手与至少 30 分钟长连接稳定性。
- Go PTY 桥接、tmux 尺寸同步、退出状态和 daemon 重启恢复。
- Mac 用户级自动启动、防睡眠操作和直接 tailnet 监听边界。

## 原型依赖候选

- 2026-09-20 从包注册表核查的当前版本：`@xterm/xterm` 6.0.0、`@xterm/addon-fit` 0.11.0、esbuild 0.28.2、`github.com/coder/websocket` 1.8.15、`github.com/creack/pty` 1.1.24。
- xterm.js 与 fit addon 的 npm 发布包均声明 MIT 许可证；其 ESM 文件分别约 345 KB 与 2 KB，可直接内嵌，不需要引入 Node 作为项目运行或构建前置。原型据此锁定 xterm.js 6.0.0 与 fit addon 0.11.0。
- `tailscale status --json` 实测 `.Peer` 为 `null`，表示 Mac 当前看不到其他 tailnet peer；手机安装完成不等于已经在相同 tailnet 在线，真机测试前必须再次确认。
- 手机打开 Tailscale 后，Mac 能看到在线 iOS peer；真实直连页面与 WebSocket 已建立。目标 Tailscale 版本为 1.102.4，首次执行 Serve 需要 tailnet 管理员通过控制台链接显式启用。

## 移动端 UI 观察

- 390×844 的真实浏览器渲染显示：当前页面功能优先但视觉层级不足；顶栏只有标题，终端与页面背景几乎没有边界，连接状态不够醒目，底部快捷键为同质灰色按钮且右侧操作需要无提示横向滚动。
- UI 方向确定为“私人远程终端仪表盘”：石墨机身、克制的青绿色连接信号、清晰的终端框架和会话元数据、接近实体键帽的快捷键工具栏。签名元素是贯穿顶部的连接信号轨道，用于表达 tailnet → Mac → agent 的活跃状态。
- Chrome headless 通过 Serve 域名截图出现 `ERR_CONNECTION_CLOSED`，但同一时刻 curl HTTPS 与自动化 WSS 探针正常；改用 daemon localhost 可正常渲染并截图。该差异只视为本机 headless/Serve 组合限制，不推翻真实 iPhone 已通过的 Serve 结果。
- 用户提供的新参考图明确了第二轮 UI 信息架构：紧凑 Header、独立 Project/Path/Model 上下文、可扫描的运行区、强存在感多行 composer、底部快捷操作。应复用其层级和触控逻辑，而不是照搬大量 glow 或伪造聊天记录。
- 当前前端仍只有单个 xterm.js 终端、连接状态和快捷键条；后端没有结构化 conversation/tool/status 数据，也没有项目、路径或模型切换 API。根据既定边界，第二轮重构继续把原生 TUI 作为权威运行现场，不解析终端文本推断消息或任务状态。
- 可立即真实化的数据只有 WebSocket 连接状态、终端尺寸、配置的 agent 与 cwd。需要新增同源只读 runtime context API，把 cwd 的 basename 作为“当前工作区”展示；模型只能显示为原生会话默认值，不能声称已读到 provider 模型。
- 第二轮 UI 的 500×1000 真实浏览器截图已复核：Header、三段上下文条、原生 TUI 卡片、composer 与四项 action bar 的层级清晰，长 cwd 正确截断，composer 固定在滚动运行区下方。首次 screenshot 因 headless Chrome 过早截取未出现 TUI 内容；增加页面等待后 tmux 历史正常重绘，WebSocket 与 xterm 没有因布局重构失效。
