# 电脑端运行时

状态：Web + Tailscale 运行合同；多终端、目录切换、显式结束、tmux 保活/恢复与浏览器附着原型已实现，创建幂等和电脑重启恢复仍待实现。产品边界见 [product.md](product.md)。

## 本地进程与环境

Daemon 以 Mac 当前用户身份运行，提供 Web 页面、控制 API 与终端流，并管理自己创建的终端。首版不接管用户已在其他终端中打开的任意进程。

Agent 沿用本机用户环境、配置与凭据。不能因为远程启动就替换 `HOME`、`CODEX_HOME`，或自动添加跳过权限确认的参数。provider 凭据不发送到浏览器或 Tailscale 控制面。

为了让 Web 终端呈现 agent 原生 ANSI 样式，新建 agent 进程时只从继承环境中移除 `NO_COLOR`。这不修改用户 shell、配置文件或全局环境；已经运行的进程不会被热修改，需要在用户明确结束并新建终端后生效。

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

网页请求目录列表或提交路径，daemon 展开当前用户的 `~`、解析绝对路径和符号链接，验证它是存在且可访问的目录，返回规范路径、父目录及最多 200 个直接子目录。目录服务不递归扫描，也不需要项目记录、绑定 API、Git 仓库标识或克隆流程。

最近使用路径可作为浏览器本地偏好保存；这只是导航捷径。目录被删除或移动后显示不可用，不能悄悄创建新目录或切到其他位置。

终端保存启动时的 `cwd`。目录内后续文件操作交给 agent，同目录的多个 agent 共享真实文件，产品不自动创建 worktree 或处理冲突。

## 终端托管

建议每个 `terminal_id` 对应产品专用 tmux server 中的一个 session，内含一个 agent pane。tmux 的 server/socket 与用户平时使用的 tmux 分开；忽略会影响显示的用户 tmux 配置，避免界面行为依赖个人设置。

创建过程：

1. Daemon 为创建请求生成 `session-<12hex>` 的 `terminal_id`，使用请求中已验证的工作目录与 daemon 配置的 `agent_id`；请求不提交目录时回退到 daemon 默认 cwd。
2. Daemon 启动独立 tmux session，以所选目录作为 `cwd` 运行发现的 CLI；总数有上限。
3. 使用参数数组及工作目录参数，不能把路径或控制字段拼成 shell 命令。用户在终端中的输入只在创建成功后作为终端字节传入。
4. 当前以产品专用 tmux socket、受限 session 名称和 `@code-remote-cwd` user option 恢复运行终端；旧 session 没有该元数据时回退到 daemon 默认 cwd。浏览器只在 localStorage 保存最近选择的 `terminal_id`，不保存正文。
5. 浏览器附着后，daemon 创建 PTY 启动 tmux attach 客户端，把终端流桥接到当前 WebSocket。

列出终端时可读取 tmux `#{pane_title}`。该字段由 pane 内应用通过标准终端标题控制序列维护；当它包含可用的 agent 原生会话名时作为展示标题返回，尚未命名时回退为“新对话”或本地编号。不得解析屏幕正文、读取 provider 会话文件或使用继承环境变量来推断标题。

Daemon 不将 agent 绑定到某个 HTTP 请求或 WebSocket 的取消上下文。连接断开时可以结束 attach 客户端，但不得销毁 tmux session。

Codex 历史列表通过本机 CLI 的官方 app-server `thread/list` 按浏览器当前终端 cwd 查询，只读取 `id/name/preview/time` 元数据。选择历史项时，Daemon 再次校验 cwd，并以参数数组在同一目录的新 tmux session 中运行 `codex resume <thread_id>`；它不解析 provider 会话文件，也不把历史复制进产品存储。

用户从手机明确结束终端时，Daemon 只接受 catalog 已知的 `terminal_id`，以参数数组精确结束对应 tmux session，成功后从运行列表移除。重复结束已经移除的 ID 是幂等成功；结束失败时保留 catalog 条目供用户重试。该操作不调用 provider 历史删除接口，因此 Codex 原生 session 仍可列出和恢复。

Codex 模型和 slash command 继续由原生 TUI 处理。网页可向当前已附着终端发送 `/model`、`/status`、`/permissions`、`/review`，或只输入 `/` 打开原生命令菜单；不读取或覆盖 Codex 配置，不自动选择模型或权限，也不对其他 agent 声称支持。

网页预组装上述 Codex 快捷指令时，以 bracketed-paste 起止序列界定正文；需要执行的指令在结束序列后追加回车，只打开菜单的 `/` 不追加。不能把正文与回车作为无边界的普通文本批次发送：Codex 0.155.1 的 `/model` 和 0.156.1 的旧 composer 中文指令验收会把这种输入留在原生输入栏而不提交。当前页面已按用户要求移除重复 composer；直接在 xterm 中键入仍按原始字节传递，不覆盖已有原生输入。

终端聚焦时，浏览器发送 `terminal.focus`。Daemon 查询所选终端唯一 pane 的 `#{pane_in_mode}`；仅值为 `1` 时执行 `send-keys -X cancel` 退出 tmux copy mode，普通输入栏和 Codex 菜单不收到额外 Esc。移动端终端滑动会释放 xterm 焦点，避免 copy mode 的定位光标被误看成原生输入光标。键盘占用视口时，页面使用 Safari `visualViewport.height` 调整原生终端可见高度。

建议让退出后的 pane 保留到用户关闭终端，以便展示最终输出和退出码；agent 已退出时显示 `exited`，不能把仍存在的 tmux pane 当作 agent 正在运行。

## 生命周期

终端进程状态使用 `starting / running / exited / failed / unknown`；附着连接状态使用 `connecting / attached / disconnected`。`running` 只表示进程存活，不能据此断言 agent 正在生成、空闲或等待批准。

| 事件 | 预期行为 |
| --- | --- |
| 网页切到另一个终端 | 脱离当前显示，两个 agent 都继续运行 |
| 锁屏、关闭页面、网络或 Tailscale 中断 | WebSocket 可以关闭，tmux 和 agent 保留 |
| 重新打开网页 | 重新加载终端列表，附着原 `terminal_id`，恢复屏幕 |
| Tailscale 重连 | 浏览器重新建立网络与 WebSocket；电脑进程不依赖手机连接存活 |
| Daemon 重启 | 专用 tmux server 若仍存活，通过 `prototype` / `session-<12hex>` 名称重新发现终端，不重复创建 |
| Agent 自己退出 | 保留可见退出结果，不自动重新发送任务 |
| 用户明确结束终端 | 经确认后仅结束指定终端及其宿主任务；不能影响同目录其他终端，也不删除 agent 原生 session |
| 电脑重启或 tmux 丢失 | 活跃终端无法直接复活；显示终端丢失，后续通过 agent 原生能力恢复历史，不承诺无损续跑 |

Daemon 恢复时以实际 tmux/进程状态核对本地元数据。启动中途失败时记录错误；不能因状态文件中有记录就报告运行成功。

## 屏幕恢复与输入

重连创建新的 `attachment_id`，重置浏览器终端解析状态，按当前视口尺寸重新附着，让 tmux 重绘当前画面。不能简单拼接历史纯文本来恢复带光标、颜色和交互菜单的 TUI。

当前屏幕恢复是首版要求；滚动历史可使用 tmux 有界历史及浏览器终端能力，但不承诺永久保留全部输出。触摸滚动、复制、中文输入、多行粘贴和特殊键必须在 iPhone Safari 验证。

专用 tmux server 开启 mouse，并将全屏 alternate-screen 下的 `WheelUpPane` 固定进入 copy mode。浏览器只在 TUI 区域把单指垂直手势编码为终端鼠标滚轮序列；这条路径查看 tmux 有界回滚，不改变 agent 输入协议或制造结构化消息。

终端输出按字节传递，不按 WebSocket 分片强行解码 UTF-8；字符与控制序列可能跨分片。尺寸变化修改附着客户端 PTY 的行列数，再由 tmux 传递终端尺寸。

tmux 附着客户端必须以全局 `-u` 选项启动，确保 daemon 在 launchd 的无 locale 环境里仍向浏览器输出 UTF-8。未启用时 tmux 可在发送前把中文替换成下划线，浏览器无法从这些字节恢复原文。

每个终端首版只允许一个可写附着。网络慢时使用有界队列；严重积压时断开显示并重新附着，不能因为浏览器无人消费输出而阻塞 agent，也不能任意丢控制序列后继续显示。

断线期间不自动排队发送输入；结果未知的 Enter、控制键或任务文本不自动重放，避免重复执行。详情见 [protocol.md](protocol.md)。
