# 电脑端运行时

状态：Web + Tailscale 运行合同；单终端 daemon、tmux 保活与浏览器附着原型已实现，目录入口、多终端与持久化恢复仍待实现。产品边界见 [product.md](product.md)。

## 本地进程与环境

Daemon 以 Mac 当前用户身份运行，提供 Web 页面、控制 API 与终端流，并管理自己创建的终端。首版不接管用户已在其他终端中打开的任意进程。

Agent 沿用本机用户环境、配置与凭据。不能因为远程启动就替换 `HOME`、`CODEX_HOME`，或自动添加跳过权限确认的参数。provider 凭据不发送到浏览器或 Tailscale 控制面。

“发现 CLI”只表示找到可执行入口，不表示已经登录、额度可用或所有运行依赖齐全。启动错误和 agent 原生认证提示照常展示，首版不自动登录或安装 provider。

## Web 服务与 Tailscale

服务只允许以下两种监听方式，具体基线由首个原型决定：

1. 直接绑定 Mac 的 Tailscale 地址，由 iPhone 通过 MagicDNS 名称或 Tailscale IP 访问。
2. 只绑定 `127.0.0.1`，由 Tailscale Serve 私有反向代理。

禁止默认监听 `0.0.0.0`、普通 LAN 地址或公网地址。禁止用 Tailscale Funnel 暴露服务。HTTP API 不开放跨域 CORS；WebSocket 必须校验 `Origin`，只接受当前 Code Remote 页面来源。

使用 Serve 身份头时，daemon 必须只监听 localhost；直接访问模式不能信任客户端自行提交的身份头。首版访问边界由 tailnet 成员关系与 ACL 提供，不另建产品账号或配对数据库。

Mac 进入睡眠、退出 Tailscale 或 daemon 停止后，手机无法访问；首版不实现远程唤醒。

## CLI 发现

首版描述表先包含 `codex`、`claude`：标识、显示名、默认命令名。统一的是终端传输，不是 provider API。

建议顺序：

1. 若本机配置了显式可执行路径，验证该路径；无效则显示配置错误，不静默换用另一份安装。
2. 否则查找 daemon 的 PATH，取得可执行入口及启动所需环境。
3. 未找到时，按需通过用户 shell 补充解析已知命令；设置超时、缓存结果，避免每次页面刷新执行 shell 初始化文件。
4. 仍未找到时显示未安装/未发现，并允许刷新；首版不扫描整块磁盘。

启动时扫描，网页可手动刷新。版本探测是独立的限时检查，失败不能伪装成“未安装”。发现与启动必须使用一致环境：仅找到绝对脚本路径而缺失它需要的 Node/PATH，仍会启动失败。

可选 provider 特定安装路径需依据真实安装样本补充。不读取各 provider 的账号 token 判断安装状态。

参考 [Multica agents_probe.go](https://github.com/multica-ai/multica/blob/main/server/internal/daemon/agents_probe.go) 与 [agents_refresh.go](https://github.com/multica-ai/multica/blob/main/server/internal/daemon/agents_refresh.go)。只借鉴发现职责，不引入 workspace 注册、模型配置或任务队列。

## 工作目录

网页请求目录列表或提交路径，daemon 验证它是存在且可访问的目录，返回规范路径及目录项。目录服务不需要项目记录、绑定 API、Git 仓库标识或克隆流程。

最近使用路径可作为浏览器本地偏好保存；这只是导航捷径。目录被删除或移动后显示不可用，不能悄悄创建新目录或切到其他位置。

终端保存启动时的 `cwd`。目录内后续文件操作交给 agent，同目录的多个 agent 共享真实文件，产品不自动创建 worktree 或处理冲突。

## 终端托管

建议每个 `terminal_id` 对应产品专用 tmux server 中的一个 session，内含一个 agent pane。tmux 的 server/socket 与用户平时使用的 tmux 分开；忽略会影响显示的用户 tmux 配置，避免界面行为依赖个人设置。

创建过程：

1. 浏览器为一次创建意图生成固定 `terminal_id`，发送目录、`agent_id`、初始行列数。
2. Daemon 验证请求、目录和 agent，记录创建意图，启动独立 tmux session，以所选目录作为 `cwd` 运行发现的 CLI。
3. 使用参数数组及工作目录参数，不能把路径或控制字段拼成 shell 命令。用户在终端中的输入只在创建成功后作为终端字节传入。
4. 将终端标识、目录、agent、创建时间存入本地状态，并在 tmux 元数据上保留恢复标识。相同创建意图重试不得启动第二个 agent。
5. 浏览器附着后，daemon 创建 PTY 启动 tmux attach 客户端，把终端流桥接到当前 WebSocket。

Daemon 不将 agent 绑定到某个 HTTP 请求或 WebSocket 的取消上下文。连接断开时可以结束 attach 客户端，但不得销毁 tmux session。

建议让退出后的 pane 保留到用户关闭终端，以便展示最终输出和退出码；agent 已退出时显示 `exited`，不能把仍存在的 tmux pane 当作 agent 正在运行。

## 生命周期

终端进程状态使用 `starting / running / exited / failed / unknown`；附着连接状态使用 `connecting / attached / disconnected`。`running` 只表示进程存活，不能据此断言 agent 正在生成、空闲或等待批准。

| 事件 | 预期行为 |
| --- | --- |
| 网页切到另一个终端 | 脱离当前显示，两个 agent 都继续运行 |
| 锁屏、关闭页面、网络或 Tailscale 中断 | WebSocket 可以关闭，tmux 和 agent 保留 |
| 重新打开网页 | 重新加载终端列表，附着原 `terminal_id`，恢复屏幕 |
| Tailscale 重连 | 浏览器重新建立网络与 WebSocket；电脑进程不依赖手机连接存活 |
| Daemon 重启 | 专用 tmux server 若仍存活，通过本地状态及 tmux 元数据重新发现终端，不重复创建 |
| Agent 自己退出 | 保留可见退出结果，不自动重新发送任务 |
| 用户明确结束终端 | 仅结束指定终端及其宿主任务；不能影响同目录其他终端 |
| 电脑重启或 tmux 丢失 | 活跃终端无法直接复活；显示终端丢失，后续通过 agent 原生能力恢复历史，不承诺无损续跑 |

Daemon 恢复时以实际 tmux/进程状态核对本地元数据。启动中途失败时记录错误；不能因状态文件中有记录就报告运行成功。

## 屏幕恢复与输入

重连创建新的 `attachment_id`，重置浏览器终端解析状态，按当前视口尺寸重新附着，让 tmux 重绘当前画面。不能简单拼接历史纯文本来恢复带光标、颜色和交互菜单的 TUI。

当前屏幕恢复是首版要求；滚动历史可使用 tmux 有界历史及浏览器终端能力，但不承诺永久保留全部输出。触摸滚动、复制、中文输入、多行粘贴和特殊键必须在 iPhone Safari 验证。

终端输出按字节传递，不按 WebSocket 分片强行解码 UTF-8；字符与控制序列可能跨分片。尺寸变化修改附着客户端 PTY 的行列数，再由 tmux 传递终端尺寸。

每个终端首版只允许一个可写附着。网络慢时使用有界队列；严重积压时断开显示并重新附着，不能因为浏览器无人消费输出而阻塞 agent，也不能任意丢控制序列后继续显示。

断线期间不自动排队发送输入；结果未知的 Enter、控制键或任务文本不自动重放，避免重复执行。详情见 [protocol.md](protocol.md)。
