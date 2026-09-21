# Code Remote

在 iPhone Safari 中连接自己的 Mac，选择工作目录，打开 Codex / Claude 等 agent 的原生 TUI。

首版面向个人自用：iPhone 与 Mac 通过 Tailscale 私有网络连接，Mac 上的 Web 服务负责目录、终端和 TUI 数据，tmux 让手机断线后任务继续运行并可重新附着。没有公网 Relay、多 agent 协同、聊天数据库或项目管理系统。

## 当前状态

产品方向已于 2026-09-20 确认为 Web + Tailscale。仓库已有可编译的 Go 占位入口，但 Web UI、HTTP/WebSocket 服务、终端桥接和 Tailscale 集成均尚未实现或端到端验证。

建议技术基线是 Go 本地服务、浏览器终端组件、tmux 和 Tailscale；Mac 是首个电脑平台，iPhone Safari 是首个手机客户端。Tailscale Serve 的 WebSocket 兼容性必须通过原型验证，不能视为已跑通。

## 文档入口

| 文档 | 用途 |
| --- | --- |
| [architecture.md](architecture.md) | 组件职责、数据流、建议技术栈与代码目录 |
| [产品范围](docs/product.md) | 用户确认的首版需求与交互边界 |
| [决策记录](docs/decisions.md) | 确认事项、工程建议、被取代的决策与未验证假设 |
| [电脑运行时](docs/runtime.md) | CLI 发现、目录、终端进程与恢复 |
| [通信协议](docs/protocol.md) | HTTP/WebSocket、终端流与重连约定 |
| [验证与开发顺序](docs/validation.md) | 最小原型、iPhone 验收与技术风险 |
| [开发环境](docs/development.md) | Go、tmux、Tailscale 与常用命令 |
| [AGENTS.md](AGENTS.md) | 后续 coding agent 的阅读入口和范围约束 |

`findings.md` 保留仍适用于当前方案的访谈与研究结论。`task_plan.md` 和 `progress.md` 记录阶段与验证结果；正式需求以 `docs/product.md` 为准。
