# 决策与项目记忆

更新于 2026-09-20。用户确认的需求优先于工程建议；建议未通过验证时可替换实现，不默认扩大功能范围。

## 用户已确认

| 决策 | 原因 / 原话含义 |
| --- | --- |
| Web + Tailscale | 用户在了解网页部署成本及 Tailscale 的跨网能力后明确表示“我们就用这个方案” |
| iPhone Safari 作为客户端 | Web UI 从 Mac 私有提供，不需要单独部署手机客户端 |
| 首版专注连接、运行、显示与输入 | 只要求能从手机连接和操作 agent，不做协同产品 |
| 原生 agent TUI | 复用 agent 的交互及 session，避免另建聊天层 |
| project 仅是工作目录入口 | 不新增项目注册、关联或 Project 实体 |
| 首版必须跨网 | 由 Tailscale 满足，不退回同一 Wi-Fi 限制 |
| 手机中断不停止任务，回来能恢复 | tmux 在 Mac 上保活，浏览器重新附着 |
| 同目录可开多个独立终端 | 允许切换，不要求协调文件冲突 |
| 账号业务暂缓 | 不设计统一模型服务或 provider 登录流程 |

## 建议作为实现基线

| 建议 | 理由 | 尚需验证 |
| --- | --- | --- |
| Go daemon 内嵌 Web UI | 单个本地进程便于自用安装和更新；不需要独立网页托管 | 静态资源构建方式、macOS 用户级启动 |
| 浏览器终端组件 | 复用 ANSI/光标/尺寸处理 | iPhone Safari 中文输入、特殊键、滚动、粘贴和 TUI 重绘 |
| Tailscale tailnet | 免去自建 Relay、NAT 穿透与公网暴露 | 蜂窝网络、切网、长连接和 Mac 睡眠行为 |
| 优先测试 Tailscale Serve | 可把 localhost 服务以私有 HTTPS URL 暴露并使用 ACL | Safari WebSocket 兼容性；失败时采用直接 tailnet HTTP/WS |
| tmux 托管每个独立终端 | 复用断线保活、重新附着与屏幕状态 | PTY 桥接、刷新尺寸和关闭语义 |
| tailnet 成员关系 + ACL 作为自用授权边界 | 首版只有用户自己的设备，不再建设配对系统 | 确保服务不监听普通 LAN/公网；未来多用户需重审 |
| 启动扫描 + 手动刷新 CLI | 保持首版简单，发现与版本诊断分离 | 后台进程与交互 shell 的 PATH 差异 |

## 被取代的决策

- **单实例公网 WSS Relay + 设备配对**：随 Tailscale 方案被取代；现有 relay 入口只是旧占位程序。
- **先交付同 Wi-Fi 产品**：仍不采用；Tailscale 必须在真实蜂窝网络/异地网络中验收。
- **产品管理自己的聊天会话**：不采用，继续使用 agent 原生 TUI/session。
- **创建 Project 后绑定目录**：不采用，目录仅是启动位置。
- **首版预留协同界面**：不采用，不因未来可能性扩大范围。

## 概念辨析

Tailscale 是网络接入层，不是 Code Remote 的应用后端。它解决手机与 Mac 的私网可达、加密和设备成员关系；目录浏览、终端权限、进程保活、重连状态和 UI 都仍由本项目实现。

多 provider 在首版仅指同一网页可打开不同 CLI，不表示共用账号、模型额度或对话上下文。多终端仅指独立并行进程，不保证同目录文件修改无冲突。

agent 原生 session、tmux 托管终端、浏览器 WebSocket 附着是三种不同生命周期。原生 session 可恢复历史，不自动替代存活进程或终端屏幕恢复。

“后台一直跑”落实为 Mac 持续执行。iPhone 锁屏后连接可以断开，回到页面时重新附着；不依赖 Safari 在后台常驻。

## 资料依据与限制

| 依据 | 支持的结论 |
| --- | --- |
| [Tailscale Serve](https://tailscale.com/docs/features/tailscale-serve) | 可将本机 localhost 服务通过 tailnet 私有提供，并应用 ACL/身份头 |
| [Tailscale HTTPS](https://tailscale.com/docs/how-to/set-up-https-certificates) | tailnet 可使用 MagicDNS 与 HTTPS；设备域名会进入证书透明度日志 |
| [Tailscale macOS CLI](https://tailscale.com/docs/reference/tailscale-cli?tab=macos) | macOS 客户端包含 CLI；App Store 版本使用应用内可执行文件 |
| [Serve WebSocket issue #20882](https://github.com/tailscale/tailscale/issues/20882) | 2026 年 Safari/HTTP2 WebSocket 握手存在未关闭报告 |
| [Serve WebSocket issue #18827](https://github.com/tailscale/tailscale/issues/18827) | 长连接稳定性存在未关闭报告，因此必须实测 |
| [tmux 官方指南](https://github.com/tmux/tmux/wiki/Getting-Started) | 服务端托管终端，客户端可以脱离和重新附着 |
| [Multica 发现代码](https://github.com/multica-ai/multica/blob/main/server/internal/daemon/agents_probe.go) | 显式路径、PATH 和 shell 回退可作为 CLI 发现参考 |

这些资料只证明候选能力和已知风险，不代表本项目已经集成。当前没有 Web UI、Tailscale 联调、真实 iPhone Safari 终端测试或部署服务。

## 实现前需处理

- 选定浏览器终端组件，并记录锁定版本。
- 在目标 Mac 与 iPhone 上验证直接 tailnet HTTP/WS 和 Tailscale Serve HTTPS/WSS，特别是 Safari 的 WebSocket 握手与长连接。
- 确定 daemon 的监听边界，证明普通 LAN 与公网不能访问。
- 验证路径解析后的 CLI 能在 daemon 环境中运行，特别是依赖 Node 的启动器。
- 验证 tmux 重新附着、终端尺寸、中文输入、特殊键和触摸滚动。
- 确定 Mac 防睡眠与用户级自动启动的明确操作；不把它误写成已实现的远程唤醒。
