# 开发环境

状态：Go/tmux 骨架已建立；Tailscale 已安装并登录，Web 功能尚未实现。

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

## 当前骨架

- `cmd/daemon`：当前只打印版本/占位状态；目标是 Mac Web 服务。
- `internal/protocol`：旧 v1 envelope 类型；需按 Web 协议调整。
- `cmd/relay`：被取代的公网 Relay 占位入口，不属于当前首版。

骨架不代表 Web UI、终端、Tailscale、恢复或安全边界已经实现。下一步严格按 [validation.md](validation.md) 完成最小 iPhone Safari 终端原型。
