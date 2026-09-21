# Code Remote

Code Remote 让你在外面用 iPhone Safari 连接自己的 Mac，在指定工作目录里操作 Codex 或 Claude Code 的原生终端界面。

它不是云端代跑服务。模型账号、代码、凭据和 agent 进程都留在 Mac 上；Tailscale 只负责让 iPhone 和 Mac 通过加密私网互相访问，tmux 负责在网页断开后继续保留终端。

```text
iPhone Safari → Tailscale 私网 → Mac 上的 Code Remote → tmux → Codex / Claude Code
```

## 现在能做什么

当前仓库提供的是可运行的多终端原型：

- 从 iPhone Safari 打开 Mac 上的私有 Web 页面。
- 在指定目录启动 Codex 或 Claude Code，并显示其原生 TUI。
- 从手机输入任务、查看输出和响应 agent 自己的确认提示。
- 在手机上新建、列出和切换多个独立终端；列表优先显示 agent 原生会话标题，切走后其他 agent 继续运行。
- 在手机上手输或逐层浏览工作目录，并在所选目录启动新的独立终端；当前终端不会被改 cwd 或结束。
- Codex 模式下列出当前目录的已保存对话，显示 Codex 自己的名称与预览，并可用原生 `codex resume` 恢复到新的独立终端。
- 从模型面板打开 Codex 原生 `/model` 选择器，也可打开原生 `/` 菜单或快捷触发 `/status`、`/permissions`、`/review`。
- 在 iPhone 的原生 TUI 区域单指上下滑动，查看 tmux 保留的有限终端历史。
- Safari 锁屏、关闭页面或临时断网后，Mac 上的 tmux 和 agent 不会因为网页断开而自动结束。
- Go daemon 重启后从产品专用 tmux server 恢复终端列表。
- 支持直接通过 Tailscale 地址访问，也可通过 Tailscale Serve 测试私有 HTTPS。

目前还不能在网页里切换 agent 或结束终端；创建请求幂等、电脑重启恢复也尚未实现。iPhone 中文输入、真机目录/模型切换、重连和长连接场景仍在继续验收，因此它还不是完成版产品。

## Codex 和 Claude 已经能远程使用，为什么还要做 Code Remote

Codex 和 Claude 各自有远程使用方式，但入口、会话和运行环境彼此独立。实际工作中，不同任务可能适合不同的模型与 agent harness；从工具调用和权限流程，到上下文管理与原生 TUI，也各有取舍。Code Remote 把选择放在终端这一层，让同一个手机入口可以连接不同 agent，同时保留它们原本的能力和交互，不把所有 provider 压成一套功能较少的通用聊天协议。

- **一个入口切换不同 agent**：不用因为任务改用另一个模型或 harness，就连同远程入口一起更换。
- **直接使用真实开发环境**：agent 运行在 Mac 的工作目录中，继续使用本机已有代码与工具链，以及原有配置和 provider 凭据，不需要把项目同步到另一套云端工作区。
- **手机连接不决定任务寿命**：tmux 托管终端，Safari 关闭、锁屏或断网只会断开显示，Mac 上的任务可以继续。
- **保留原生能力**：Codex 和 Claude 仍运行自己的 CLI 和 TUI，以及原生 session 与权限确认流程。Code Remote 只负责连接和终端管理，不尝试把两者翻译成同一种对话产品。
- **daemon 可以部署在不同 Mac 上**：任何满足依赖和网络条件的 Mac 都能运行自己的 Code Remote 节点，不要求所有人把工作集中到一台固定电脑或一个厂商的 remote app。

这里没有绕过认证。Code Remote 省掉的是再建设一套产品账号、设备配对和 provider 登录向导：访问者仍需通过 Tailscale 身份与 ACL 进入对应 tailnet，Codex 或 Claude 也继续使用 Mac 上已登录的账号，并保留原生权限确认。

这套方式可以服务彼此信任的小团队。每个人可以在自己的 Mac 上部署 daemon，也可以让获准成员访问一台共用开发机。不过当前版本以启动 daemon 的同一个 macOS 用户身份运行，没有按访问者隔离目录和 agent 凭据，也没有独立的终端权限或审计记录；同一目录的并行终端还会共享真实文件。访问规模也受 tailnet 权限配置及所用 Tailscale 套餐约束。它适合小范围、明确授权的合作，不能当成公开服务或完整的多租户协作平台。

## Mac 和 iPhone 需要安装什么

| 设备 | 软件或条件 | 用途 |
| --- | --- | --- |
| Mac | [Tailscale](https://tailscale.com/download/mac) | 把服务安全地放在自己的 tailnet 中 |
| iPhone | [Tailscale](https://apps.apple.com/app/tailscale/id1470499037) | 让 Safari 跨网络访问 Mac |
| Mac | [Homebrew](https://brew.sh/) | 推荐的依赖安装方式 |
| Mac | Go 与 tmux | 构建 daemon，并在断线后保留终端；可由 `make bootstrap` 安装 |
| Mac | [Codex CLI](https://developers.openai.com/codex/cli/) 或 [Claude Code](https://docs.anthropic.com/en/docs/claude-code/setup) | 至少安装并登录一个要远程使用的 agent |
| Mac | Git | 下载和更新本仓库 |

Code Remote 不会替你安装 agent、登录 provider、复制凭据，也不会修改 `HOME`、`CODEX_HOME` 或自动跳过 agent 的权限确认。

## 快速开始

### 1. 准备 Tailscale

1. 在 Mac 和 iPhone 上安装 Tailscale。
2. 两台设备登录同一个 tailnet，并在 iPhone 上开启 Tailscale VPN。
3. 确认 Mac 和 iPhone 在 Tailscale 中都显示在线。
4. 如果 tailnet 使用 ACL，只允许自己的 iPhone 或用户访问 Code Remote 使用的端口。

不要启用 Tailscale Funnel。Funnel 会把服务开放到公网，不在本项目的安全边界内。

### 2. 下载并构建 Code Remote

```sh
git clone https://github.com/huangxinxinyu/CodeRemote.git
cd CodeRemote

make bootstrap
make doctor
make build
```

`make bootstrap` 通过 Homebrew 安装缺失的 Go 和 tmux，不会安装或升级 Tailscale，也不会安装 Codex 或 Claude Code。如果不想使用 Homebrew，可以自行安装 Go 与 tmux，然后从 `make doctor` 开始。

`make doctor` 应能找到 Go、tmux、Tailscale，以及你准备使用的 agent。首次远程运行前，先在 Mac 的普通终端里启动一次 `codex` 或 `claude`，完成各自的登录和初始化。

构建完成后，可执行文件位于：

```text
bin/code-remote-daemon
```

### 3. 选择接入方式并启动

当前有两种私网接入方式。建议先试直接访问；如果需要 Safari 的 HTTPS 页面，再测试 Tailscale Serve。Serve 的 WebSocket 长连接兼容性仍需在你的设备上验证。

#### 方式 A：直接使用 Tailscale 地址

先查看 Mac 的 Tailscale IPv4 地址：

```sh
tailscale ip -4
```

然后启动 daemon。把示例 IP 和工作目录换成自己的真实值：

```sh
./bin/code-remote-daemon \
  -listen 100.78.102.15:8080 \
  -agent codex \
  -cwd /absolute/path/to/workspace
```

在已连接同一 tailnet 的 iPhone Safari 中打开：

```text
http://100.78.102.15:8080
```

只能填写这台 Mac 自己的 Tailscale IP。daemon 会拒绝普通局域网地址以及 `0.0.0.0` 这类全网卡监听地址。

如果使用 Mac App Store 版 Tailscale，CLI 没有出现在 `PATH` 中，可以这样查询地址：

```sh
TAILSCALE_BE_CLI=1 /Applications/Tailscale.app/Contents/MacOS/Tailscale ip -4
```

#### 方式 B：通过 Tailscale Serve 使用 HTTPS

daemon 必须只监听本机回环地址：

```sh
./bin/code-remote-daemon \
  -listen 127.0.0.1:8080 \
  -agent codex \
  -cwd /absolute/path/to/workspace
```

另开一个 Mac 终端执行：

```sh
tailscale serve --bg 8080
tailscale serve status
```

在 iPhone Safari 中打开 `tailscale serve status` 显示的私有 `https://...ts.net` 地址。首次使用 Serve 时，Tailscale 可能要求 tailnet 管理员启用 HTTPS。

Serve 只能作为 tailnet 内的私有入口，不要把 `serve` 换成 `funnel`。

## 启动参数

| 参数 | 默认值 | 说明 |
| --- | --- | --- |
| `-listen` | `127.0.0.1:8080` | 只能使用 loopback 或这台 Mac 的 Tailscale 地址 |
| `-agent` | `codex` | 当前只接受 `codex` 或 `claude` |
| `-cwd` | 启动命令时的当前目录 | agent 的工作目录；必须是已存在的目录 |
| `-version` | 关闭 | 显示 daemon 版本后退出 |

例如，要在另一个目录中运行 Claude Code：

```sh
./bin/code-remote-daemon \
  -listen 127.0.0.1:8080 \
  -agent claude \
  -cwd "/Users/your-name/Developer/my project"
```

路径可以包含空格，但必须使用引号包住。`-cwd` 是首次打开及旧 session 的默认目录；网页可在其他已存在目录新建终端。Agent 仍由 daemon 启动参数决定，切换 agent 需要停止 daemon 后用新的参数重新启动。

## 使用与停止

保持 Mac 开机、联网、连接 Tailscale，并避免进入会中断进程或网络的睡眠状态。Code Remote 不提供远程开机或唤醒。

打开网页后即可像本地终端一样操作 agent。关闭 Safari 页面只会断开显示；固定名为 `prototype` 的专用 tmux session 和其中的 agent 会继续运行。再次打开同一地址时，daemon 会重新附着该终端。

在 Mac 上按 `Ctrl+C` 会停止 Web daemon，但不会承诺终止 tmux 中已在运行的 agent。要结束 agent，优先在其原生 TUI 中正常退出。不要直接删除 tmux socket 或状态文件来代替正常退出。

## 常见问题

### `make doctor` 显示 missing

根据输出安装缺失的软件。`make bootstrap` 只处理 Go 和 tmux；Tailscale 与 Codex/Claude Code 需要单独安装。

### iPhone 打不开页面

- 确认两台设备登录同一 tailnet，iPhone 上的 Tailscale VPN 已开启。
- 直接访问时，确认 `-listen` 和 Safari URL 使用同一个 Mac Tailscale IP。
- Serve 模式下，确认 daemon 监听 `127.0.0.1:8080`，并检查 `tailscale serve status`。
- 检查 tailnet ACL 和 macOS 防火墙是否允许目标连接。
- 确认 Mac 没有睡眠，daemon 仍在运行。

### Serve 页面能打开，但终端容易断开

这是当前仍在验证的组合。先改用“方式 A”的直接 tailnet HTTP/WebSocket，记录 macOS、iOS、Safari 和 Tailscale 版本，再对照[真实设备验收表](docs/validation.md)排查。

### daemon 找不到 `codex` 或 `claude`

先在启动 daemon 的同一个 Mac 终端运行：

```sh
command -v codex
command -v claude
```

daemon 目前从自己的 `PATH` 查找 agent。某个 CLI 只在交互式 shell 初始化后才可见时，应先修正启动环境的 `PATH`。

### 网页断开后任务还会继续吗

网页、WebSocket 或 iPhone 网络断开不会默认结束 tmux 中的 agent。但 Mac 关机、睡眠、退出 Tailscale、tmux server 消失或机器重启不属于无损续跑保证。

## 开发

```sh
make test    # 运行 Go 测试
make check   # 检查 gofmt、Go test/vet 和开发环境脚本
make build   # 构建 bin/code-remote-daemon
```

更详细的产品和技术合同见：

- [产品范围](docs/product.md)
- [项目架构](architecture.md)
- [电脑运行时](docs/runtime.md)
- [通信协议](docs/protocol.md)
- [开发环境](docs/development.md)
- [验证与真实设备验收](docs/validation.md)
- [决策记录](docs/decisions.md)

## 安全边界

- 服务只应监听 localhost 或 Mac 的 Tailscale 地址。
- 不监听 `0.0.0.0`，不把端口暴露到普通 LAN 或公网。
- 不启用 Tailscale Funnel。
- HTTP API 保持同源，WebSocket 会校验 `Origin`。
- 终端可能显示代码和凭据；不要把终端输出、provider token 或 Tailscale 身份信息写入公开日志或提交到仓库。

Code Remote 首版是个人自用工具，不提供多用户权限、公开部署、产品账号或聊天数据库。
