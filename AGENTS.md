# 项目接续指南

## 阅读顺序

先读 [README.md](README.md)、[产品范围](docs/product.md)、[architecture.md](architecture.md) 与 [决策记录](docs/decisions.md)。涉及电脑执行或通信时再读 [runtime.md](docs/runtime.md)、[protocol.md](docs/protocol.md)，环境准备见 [development.md](docs/development.md)，实施与验收参考 [validation.md](docs/validation.md)。

## 产品边界

- 首版是 iPhone Safari Web UI，通过 Tailscale 私有连接 Mac 上的 agent 原生 TUI，不自建公网 Relay。
- 跨网、多个独立终端、手机断线后电脑继续运行及重连恢复仍在范围内；不能退回同一 Wi-Fi 产品。
- project/workspace 只是工作目录入口，不新增项目注册/绑定业务或 Project 实体。
- 不新增聊天数据库、跨 provider 对话协议、多 agent 协同、任务调度、自动 worktree 或模型账号体系。
- agent 原生 session 与产品终端 ID、浏览器附着 ID 分开。WebSocket 断开或页面离开不能默认结束 agent。
- 多终端同目录意味着共享文件，不承诺自动解决冲突。

## 技术与证据

目标基线是 Go Web daemon + 浏览器终端组件 + tmux + Tailscale，macOS 与 iPhone Safari 优先。Tailscale Serve 的 WebSocket 兼容性有公开未关闭问题，必须与直接 tailnet HTTP/WS 一起原型验证，不能描述为已经跑通。

仓库已有 Go daemon 与旧 relay 的可编译占位骨架。`cmd/relay` 不属于当前首版，不要把占位入口描述为已实现服务，也不要声称已有 iPhone、Tailscale 或端到端测试。

服务不得默认监听全部网卡、普通 LAN 或公网；不启用 Tailscale Funnel。使用 Serve 时后端只监听 localhost；直接模式只绑定 Tailscale 地址。WebSocket 校验 Origin，HTTP API 不开放宽泛 CORS。

复用电脑已有 agent 环境，不覆盖用户的 HOME/CODEX_HOME，不自动增加跳过审批的参数。首版不解析 provider 会话文件，不依据终端文本推断结构化任务状态。

修改技术方案时同步更新 architecture.md 与 docs/decisions.md，写明依据；不能以工程方便为由静默改成其他客户端形态、公网 Relay 或同局域网产品。新增功能需有用户需求，不因未来可能协同而预先扩建。

## Memory 维护

正式需求以 docs/product.md 为准，决策及假设以 docs/decisions.md 为准，技术合同在 runtime.md/protocol.md。更新时区分用户确认、工程建议、资料核实和实际测试结果。

findings.md 保留仍适用于当前方案的访谈与研究。task_plan.md/progress.md 记录产品方向与验证结果；后续实现应新增阶段，不覆盖当前基线。
