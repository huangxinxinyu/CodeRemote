# 通信协议草案

状态：Web + Tailscale v0 合同；多终端 attach/input/output/resize、目录浏览与按 cwd 创建、Codex 历史/恢复均已在原型实现，完整错误码、关闭与创建幂等仍待实现。

## 连接与信任

iPhone Safari 只通过 Tailscale tailnet 访问 Mac daemon。首版没有公网 Relay、产品账号、设备配对或匿名公网入口。

允许的接入形式：

- Daemon 绑定 Mac 的 Tailscale 地址，提供同源 HTTP + WebSocket。
- Daemon 只绑定 `127.0.0.1`，由 Tailscale Serve 提供私有 HTTPS + WSS；是否可作为默认路径取决于真实 Safari 长连接验证。

禁止默认绑定全部网卡、开放普通 LAN 入口或启用 Tailscale Funnel。Tailscale ACL 应只允许用户预期的手机访问该 Mac 服务端口。

所有 HTTP API 采用同源调用，不返回宽泛 CORS 许可。WebSocket 服务校验 `Origin`，拒绝非 Code Remote 页面来源，避免其他网页借用已加入 tailnet 的浏览器调用终端 API。不得从查询字符串接收 bearer token、终端输入或其他敏感信息。

使用 Tailscale Serve 身份头时，只有来自 localhost 代理的头可受信；直接访问模式忽略这些头。终端输入输出正文、Tailscale 身份信息和 provider 凭据不得写入访问日志。

## 最小资源

| 资源 | 必要信息 |
| --- | --- |
| Agent | `agent_id`、显示名、是否发现、可选版本、诊断状态 |
| Directory | 已实现；规范路径、父目录、最多 200 个直接子目录项；只按需读取 |
| Terminal | `terminal_id`、`agent_id`、启动 `cwd`、可选原生 `title`、进程状态、创建时间 |
| Attachment | 临时 `attachment_id`、`terminal_id`、当前行列数 |

直连 Mac 后不需要 Relay 路由用的 `host_id`。没有 Project、Task、ChatMessage 或跨 provider Conversation 资源。

## HTTP 与 WebSocket

静态 Web UI 和低频控制操作使用版本化 HTTP API；附着终端使用 WebSocket。具体 URL 在原型中固定，建议形态如下：

| 操作 | 建议接口 | 语义 |
| --- | --- | --- |
| 当前上下文 | `GET /api/v1/context` | 已实现；返回当前原型的 `agent_id`、`working_directory`、派生 `workspace_name`、`terminal_id` 与 `native session` 模型标记；只读 |
| 查看/刷新 agent | `GET/POST /api/v1/agents` | 读取或重新发现本机 CLI |
| 浏览目录 | `GET /api/v1/directories?path=...` | 已实现；展开 `~` 并返回规范路径、父目录与有界的直接子目录，不递归扫描 |
| 终端列表 | `GET /api/v1/terminals` | 已实现；返回专用 tmux server 中恢复的终端上下文及可选 `title`；标题来自 pane title 元数据，不含正文 |
| 创建终端 | `POST /api/v1/terminals` | 已实现；可提交 `working_directory`，服务端验证后生成 ID 并在该 cwd 启动，不创建聊天记录 |
| Codex 历史 | `GET /api/v1/codex/threads?working_directory=...` | Codex 模式已实现；返回当前终端 cwd 的原生 `name`、`preview` 与时间，不读取 provider 文件 |
| 恢复 Codex 对话 | `POST /api/v1/codex/threads/{id}/resume` | Codex 模式已实现；请求提交当前 cwd，重新核对原生历史 ID 后在该目录的新 tmux 终端运行 `codex resume` |
| 关闭终端 | `DELETE /api/v1/terminals/{id}` | 尚未实现；用户明确结束指定终端，与断开页面严格分开 |
| 附着终端 | `WS /api/v1/terminals/{id}/attach` | 已实现；附着已知终端，尺寸在首条消息内发送，不依赖查询参数 |

创建与修改请求使用 JSON，必须限制 body 大小并拒绝未知控制字段。启动命令由 daemon 的 agent 描述表决定，远程 API 不接受任意 shell 命令字符串。

按目录创建的请求体为 `{"working_directory":"~/Developer/project"}`；省略字段时使用 daemon 默认 cwd。路径不存在、不是目录或不可访问时返回 400，不能自动创建或静默替换为其他目录。Codex slash command 不新增 HTTP 资源，它们作为当前附着的普通终端输入交给原生 TUI。

## WebSocket 消息

首版建议使用 JSON envelope；终端数据使用 base64 保存原始字节，先确保正确性，再根据测量结果决定是否引入二进制帧。

```json
{
  "v": 1,
  "type": "terminal.attach",
  "request_id": "request-uuid",
  "payload": {
    "cols": 60,
    "rows": 30
  }
}
```

服务端确认附着后返回新的 `attachment_id`。后续消息：

| 消息 | 方向 | 语义 |
| --- | --- | --- |
| `terminal.attached` | 服务端 → 浏览器 | 附着成功及 `attachment_id` |
| `terminal.input` | 浏览器 → 服务端 | `attachment_id` + `data_base64`，写入 PTY |
| `terminal.output` | 服务端 → 浏览器 | `attachment_id` + 流内序号 + `data_base64` |
| `terminal.resize` | 浏览器 → 服务端 | 更新该附着的行列数 |
| `terminal.exited` | 服务端 → 浏览器 | agent 已退出及可取得的退出码 |
| `terminal.error` | 服务端 → 浏览器 | 结构化附着或传输错误 |

终端内输入保留用户自己的 agent 所允许的能力，不宣称目录沙箱。服务端必须验证消息版本、类型、附件归属、字段长度和尺寸上限。

## 重连与幂等

1. 页面加载后重新获取终端列表，浏览器本地只保留此前选择的 `terminal_id`。
2. 若该终端仍存在，建立新 WebSocket 并发送 `terminal.attach`，不发送新的创建请求。
3. 服务端返回新的 `attachment_id` 后才开始输出；浏览器先重置终端显示状态。
4. 新附着建立后，旧附着的输入和输出失效。首版每终端只有一个可写附着，daemon 原子替换旧连接。
5. 输出序号只在一个附着内有效；发生缺口或过量积压时重新附着和重绘，不继续拼接残缺 ANSI 流。

当前创建请求由服务端生成 `terminal_id`，尚未实现幂等键。浏览器收到成功响应后将 ID 加入列表并记为当前项；若响应结果未知，应先重新获取列表，不能盲目 POST。后续为创建意图增加幂等键后，才可安全自动重试。

关闭已关闭的终端可返回已关闭。离线时不排队创建、关闭或输入；请求结果未知时先查询状态，不盲目重发。

键盘输入没有应用级重放保证：连接中断时可能无法判断最后一段是否已执行。页面显示连接中断，重连后由用户根据屏幕继续。

## 背压与失败

浏览器使用带退避的重连；Tailscale 自身的链路变化不改变应用级重连语义。帧大小、目录分页、请求 body 和每连接队列必须有限，具体阈值由原型测量后固定。

终端桥接队列溢出时断开附着并报告需要重连，tmux 中的 agent 继续运行。不能为了保持 WebSocket 而让输出阻塞 agent。

最少区分：来源不允许、Tailscale/服务不可达、agent 未找到、目录不可访问、启动失败、终端已退出或丢失、附着过期、协议不兼容。不能从 TUI 文本推断任务是否完成。
