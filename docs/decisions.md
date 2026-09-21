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
| 移动端控制台采用清晰的上下文、运行现场和独立 composer | 用户要求优先解决输入区域、project/path context、status 与 next action 的信息层级，同时保留现有远程终端能力 |
| 不保留额外终端控制键条 | 用户确认手机键盘已经满足需要；界面把空间优先留给 agent 实时输出 |
| 恢复 agent 原生 ANSI 样式 | 浏览器 xterm 支持颜色与字重；新 agent 进程不继承宿主启动环境中的 `NO_COLOR`，但不改用户配置或 provider 环境目录 |
| 手机可以新建和切换对话 | “新对话”严格映射为新的 `terminal_id`、tmux session 和 agent 原生 session；旧终端继续运行，不增加产品聊天数据库 |
| 会话列表优先显示 agent 原生标题 | 读取 agent 主动写入的 tmux pane title；不解析终端正文，不读取 provider 私有会话文件，空标题仍回退为本地编号 |
| 手机能选择过去的 Codex 对话 | 通过官方 app-server `thread/list` 获取 Codex 原生名称和 ID，选择后以 `codex resume <id>` 启动新终端；不创建聊天数据库 |
| Codex 工作区支持单指上下滑动 | 触摸手势映射为隔离 tmux server 的原生滚轮/copy-mode 回滚，不把 TUI 输出复制成普通文本列表 |
| 手机可以切换工作目录 | 路径面板支持手输和逐层浏览；确认后在所选 cwd 新建独立终端，不能修改运行中终端的 cwd |
| 模型切换优先复用 Codex 原生指令 | `/model` 打开 Codex 自己的模型与推理强度选择器；网页不维护账号相关模型清单 |

## 建议作为实现基线

| 建议 | 理由 | 尚需验证 |
| --- | --- | --- |
| Go daemon 内嵌 Web UI | 单个本地进程便于自用安装和更新；不需要独立网页托管 | 静态资源构建方式、macOS 用户级启动 |
| xterm.js 6.0.0 + fit addon 0.11.0 | 复用 ANSI/光标/尺寸处理；发布文件可作为 ESM 随 Go 二进制内嵌，不依赖运行时 CDN | iPhone Safari 中文输入、特殊键、滚动、粘贴和 TUI 重绘 |
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

当前多终端恢复以产品专用 tmux socket 为事实源：保留历史 `prototype`，新建项使用 `session-<12hex>`，并用 session user option 保存规范 cwd；浏览器 localStorage 只保存最近选择的 `terminal_id`。POST 创建由服务端生成 ID，当前尚无幂等键，不能把未知响应的盲目重试描述为安全。

agent 原生 session、tmux 托管终端、浏览器 WebSocket 附着是三种不同生命周期。原生 session 可恢复历史，不自动替代存活进程或终端屏幕恢复。

新版 UI 中的状态仍只描述产品已知事实：WebSocket `connecting/attached/disconnected`、终端进程退出和终端尺寸。原生 TUI 是 Codex/Claude 输出的权威表示；页面不根据文字猜测 completed/running/waiting，也不建立结构化消息或工具调用协议。Project 继续只是 cwd 的显示名；路径切换只在所选目录创建新终端，不新增 Project 实体。

对话名称与任务状态分开：终端列表可以展示 CLI 通过终端标题控制序列写入、由 tmux 暴露的 `pane_title`。当前 Codex 会在生成自动标题后更新该值；Code Remote 只把它作为可选显示名，不能由名称推断执行进度。不得使用子进程继承的 `CODEX_THREAD_ID` 绑定会话，因为 daemon 从 Codex 内启动时该变量可能属于父会话。

Codex 历史选择是 provider-native integration，不是产品 Conversation 资源：daemon 为一次读取启动短生命周期本地 app-server，按当前 cwd 调用 `thread/list`；恢复前再次核对 ID，然后在独立 tmux session 中执行参数数组形式的 `codex resume <id>`。其他 provider 未接入同等官方接口时不伪造历史列表。

Codex 模型切换和 slash command 同样是原生 TUI integration：页面可发送 `/model`、输入 `/` 打开原生命令菜单，并快捷触发 `/status`、`/permissions`、`/review`。具体模型列表、推理强度、权限选择和命令可用性由当前 Codex CLI 与账号决定；Claude 模式不启用这些 Codex 专用按钮。会修改工作区文件的 `/init` 不做一键快捷入口。

Composer 发送成功后主动释放输入焦点，让 iPhone 键盘收起并恢复终端高度；这只改变浏览器布局，不改变 agent 或 tmux 生命周期。

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

这些资料只证明候选能力和已知风险。当前已完成 Web UI、Tailscale 基础联调、真实 iPhone Safari 单终端链路和自动化双终端探针；蜂窝网络、长连接与完整真机多终端交互仍未验收。

## 实现前需处理

- xterm.js 已锁定原型版本；完成真实 iPhone Safari 交互探针后再决定是否保持该版本。
- 在目标 Mac 与 iPhone 上验证直接 tailnet HTTP/WS 和 Tailscale Serve HTTPS/WSS，特别是 Safari 的 WebSocket 握手与长连接。
- 确定 daemon 的监听边界，证明普通 LAN 与公网不能访问。
- 验证路径解析后的 CLI 能在 daemon 环境中运行，特别是依赖 Node 的启动器。
- 验证 tmux 重新附着、终端尺寸、中文输入、特殊键和触摸滚动。
- 确定 Mac 防睡眠与用户级自动启动的明确操作；不把它误写成已实现的远程唤醒。
