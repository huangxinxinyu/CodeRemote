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
