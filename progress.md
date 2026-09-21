# 进度

## 2026-09-20

- 用户已完成 Tailscale 安装准备并要求开始原型开发；Mac CLI 已确认登录，地址为 tailnet 内私有地址，`make doctor` 全部通过。
- 已启动阶段 8：先建立 Git 基线与隔离工作区，再按静态终端交互、单终端链路、跨网双路径验证推进。
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
