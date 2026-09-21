# 开发环境

状态：Tailscale 已安装并登录；多终端、路径切换与 Codex 原生指令 Web/PTY/tmux 原型已实现，完整真实 iPhone 验收尚未完成。

## 当前机器

2026-09-20 检查结果：

| 工具 | 状态 |
| --- | --- |
| macOS | 26.6.2，Apple Silicon |
| Go | 1.26.4，已安装 |
| tmux | 3.7c，已通过 Homebrew 安装 |
| Codex | 0.155.1，已发现（2026-09-19 检查） |
| Claude Code | 2.1.251，已发现（2026-09-19 检查） |
| Tailscale | 已安装并登录；CLI 位于 `/usr/local/bin/tailscale` |

新机器先运行：

```sh
make bootstrap
```

该命令只安装缺失的 Go 与 tmux，不会安装或升级 Tailscale GUI；Tailscale 按下一节人工准备。

## Tailscale 准备

1. 按 [Tailscale macOS 安装说明](https://tailscale.com/docs/install/mac)在 Mac 安装客户端并登录个人 tailnet。
2. 在 iPhone 安装 Tailscale，登录同一 tailnet并开启 VPN。
3. 在 Mac 与 iPhone 上确认彼此在线；使用 iPhone 蜂窝网络验证，不以同一 Wi-Fi 代替。
4. 为 Code Remote 端口配置只允许目标设备/用户访问的 tailnet ACL。

Tailscale 推荐 macOS Standalone 版本；Mac App Store 版本同样可用。Standalone 版本可以安装 `/usr/local/bin/tailscale` CLI 启动器；App Store 版本的 CLI 位于：

```sh
TAILSCALE_BE_CLI=1 /Applications/Tailscale.app/Contents/MacOS/Tailscale status
```

不要启用 `tailscale funnel`。若原型测试 Serve，Go daemon 先只监听 localhost，例如 `127.0.0.1:8080`，再运行：

```sh
tailscale serve --bg 8080
tailscale serve status
```

上述命令是待执行步骤，不代表本机已经安装或配置 Tailscale。Serve 首次使用会要求启用 tailnet HTTPS；设备的完整 `*.ts.net` 名称会出现在公开证书透明度日志中，命名时不要包含敏感信息。

## 当前可用命令

```sh
make bootstrap  # 安装 Go/tmux，不升级已有版本
make doctor     # 检查 Go、tmux、Codex、Claude 和 Tailscale
make build      # 构建当前 daemon 到 bin/code-remote-daemon
make test   # 运行 Go 测试
make check      # gofmt、Go 测试/vet 和开发前置脚本测试
```

可通过 `make build BUILD_DIR=/path/to/output` 临时改变构建输出目录；默认不会在仓库根目录生成二进制。

## 运行多终端原型

通过 Tailscale 地址直接提供 HTTP/WS，只能填写本机实际 Tailscale IP：

```sh
./bin/code-remote-daemon \
  -listen 100.78.102.15:8080 \
  -agent codex \
  -cwd /absolute/path/to/workspace
```

通过 Tailscale Serve 测试 HTTPS/WSS 时，daemon 必须改为监听 localhost：

```sh
./bin/code-remote-daemon -listen 127.0.0.1:8080 -agent codex -cwd /absolute/path/to/workspace
tailscale serve --bg 8080
```

`-agent` 当前只接受 `codex` 或 `claude`，不接受远程提交的任意命令。关闭浏览器只关闭 tmux attach 客户端；专用 tmux session 与 agent 保留。现有 session 名为 `prototype`，手机新建项使用 `session-<12hex>`；daemon 重启会从专用 tmux socket 恢复两类名称。

## 当前实现

- `cmd/daemon`：运行内嵌页面、终端 catalog、HTTP API 与 WebSocket 桥接。
- `internal/web`：限制监听到 loopback/Tailscale 地址，校验修改 API/WebSocket 同源，提供终端 list/create/dynamic attach 并内嵌 xterm.js。
- `internal/directory`：展开 home shorthand、规范化和验证 cwd，并提供有界的直接子目录浏览。
- `internal/terminal`：以参数数组调用专用 tmux server，在所选 cwd 恢复/创建独立 session 并保存 cwd 元数据，再通过 PTY 临时附着；同一终端的新附着替换旧附着而不结束 agent。
- `internal/codex`：通过本机 Codex 官方 app-server 的 `thread/list` 读取所选 cwd 的原生名称与预览；不解析 `CODEX_HOME` 中的会话文件。
- `internal/protocol`：旧 v1 envelope 类型；需按 Web 协议调整。
- `cmd/relay`：被取代的公网 Relay 占位入口，不属于当前首版。

当前实现仍是验证原型；多个 cwd 的多终端与 daemon 重启恢复已完成自动化验证，路径面板及 Codex `/model` 等原生指令入口已接入。结束终端、创建幂等、Serve 长连接与完整安全验收尚未完成。下一步严格按 [validation.md](validation.md) 在真实 iPhone Safari 中验证。
