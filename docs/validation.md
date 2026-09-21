# 验证与开发顺序

状态：Mac 端自动化测试通过；真实 iPhone 已通过 tailnet 单终端链路。Serve API 已真实创建多终端；Yuniverse 路径创建、daemon 重启 cwd 恢复及 Codex `/model` 原生选择器已在 Mac 端实测。iPhone Safari 的目录/模型交互、多终端切换、断开恢复及 30 分钟长连接仍未完成。

## 先验证最不确定的组合

第一个原型是 iPhone Safari 浏览器终端，通过 Tailscale 连接 Mac daemon，再桥接到 tmux 中的真实 Codex/Claude。它必须能输入和显示，不能用桌面浏览器、聊天 mock、静态页面或只读日志代替。

同时比较两条接入路径：

1. Daemon 只绑定 Tailscale 地址，使用 HTTP/WS 直接访问。
2. Daemon 只绑定 localhost，使用 Tailscale Serve 的 HTTPS/WSS 访问。

记录实际 iOS、Safari、macOS、Tailscale、浏览器终端组件、tmux 和 agent CLI 版本。Tailscale Serve 的 WebSocket 已有公开兼容性报告，未通过目标设备长连接测试前不能选作默认路径。

## 实现顺序

1. **静态终端交互**：在最小 Web 页面嵌入候选终端组件，真机验证中文、特殊键、粘贴、触摸滚动、键盘尺寸变化。
2. **单终端链路**：Go HTTP/WebSocket + PTY/tmux，跑通真实 agent 的输入、重绘和尺寸变化；先在 Mac 本机浏览器定位问题。
3. **Tailscale 跨网**：iPhone 关闭 Wi-Fi，仅用蜂窝网络测试直接 tailnet 与 Serve 两条路径，持续运行并记录断线。
4. **完整首版流程**：补齐 CLI 发现、目录入口、多个独立终端及切换；不创建 Project/Chat/Task 层。
5. **恢复与安全**：锁屏、关闭页面、切网、Tailscale/daemon 重启、慢网络、Origin 校验和非 tailnet 访问验证。

首版 Go PTY 依赖、浏览器终端组件、WebSocket 库和静态资源构建方式在步骤 1–2 决定。依赖版本锁定后再记录到开发文档。

## 真实设备验收表

下表是待执行用例，不是已有测试结果。

| 用例 | 通过标准 |
| --- | --- |
| Tailscale 入网 | iPhone 与 Mac 加入同一 tailnet；手机蜂窝网络可访问，未入网设备不可访问 |
| 监听隔离 | 普通 LAN 地址和公网不能访问服务；未启用 Funnel；仅预期 tailnet 路径可达 |
| Origin 防护 | 非 Code Remote 来源的网页无法建立终端 WebSocket 或调用修改 API |
| CLI 发现 | 找到本机 Codex/Claude；缺失工具和错误显式路径可区分；刷新可识别新安装 |
| 环境一致 | 交互 shell 可用的 CLI 从 daemon 启动也能运行；依赖 Node 的入口没有 PATH 丢失 |
| 定位目录 | 能找到含空格、中文的目录并在其中启动；不存在目录不被自动创建 |
| 切换目录 | 从现有终端切到另一 cwd 后新终端使用目标路径，旧终端继续运行；daemon 重启后两者路径仍正确 |
| 输入与显示 | 中文输入法、英文、多行粘贴、退格、Enter、Tab、Esc、方向键与 Ctrl 有效 |
| 原生交互 | agent 自己的确认提示、执行输出和菜单正常显示；不自动批准 |
| 手机布局 | 键盘显示/隐藏、横竖屏和视口变化后画面继续可用，无持续错位 |
| 直接 tailnet WebSocket | 蜂窝网络下可连续操作至少 30 分钟，无异常握手或周期性断线 |
| Serve WebSocket | HTTPS/WSS 下完成同样测试；若失败，记录版本、复现步骤并不选为默认路径 |
| 多终端 | 同目录启动两个独立终端，切换后两者继续运行，输入不串台 |
| 锁屏恢复 | 受控长任务在锁屏期间由 Mac 继续执行，回来附着同一终端并看到当前现场 |
| 关闭页面恢复 | 关闭 Safari 标签页再打开，重新列出并附着原终端，不创建第二个 agent |
| 断网恢复 | 飞行模式后恢复、Wi-Fi/蜂窝切换后接回原终端，不自动重放输入 |
| 慢网与积压 | 快速输出不导致无限内存增长或 agent 被阻塞；重新附着可恢复显示 |
| Tailscale 重连 | Mac 上 agent 不中断，网络恢复后页面可继续操作 |
| Daemon 重启 | tmux 存活时可重新发现原终端；状态文件与实际进程一致 |
| 创建响应丢失 | 相同创建意图重试只产生一个终端，参数冲突明确报错 |
| 结束终端 | 离开页面不杀 agent；结束指定终端不影响另一个终端 |
| 原生 session | 使用 agent 自己的 TUI/命令入口恢复会话，不依赖产品解析会话文件 |
| 原生标题 | Codex 更新 pane title 后，终端列表显示该标题；空标题使用回退名，不能据此生成任务状态 |
| 历史选择 | Codex 历史区显示 app-server 返回的原生名称；点“恢复”后进入同一 thread 的新终端，不产生产品聊天副本 |
| 原生指令 | Codex `/model` 打开真实模型/推理强度选择器；`/` 菜单及快捷项进入原生流程，Claude 模式不误启用 |
| 触摸回滚 | 在黑色 TUI 区域上下滑可进入 tmux copy mode、查看历史并回到最新画面；手势不作为 agent 键盘输入 |
| 日志 | 日志无终端输入输出正文、provider 凭据和敏感 Tailscale 身份信息 |

自动化测试至少覆盖：CLI 发现、路径处理、创建幂等、Origin/监听边界、附着替换、关闭与离开的区别、帧与队列上限。真实 Safari 键盘、触摸和 TUI 兼容性必须保留人工验收。

## 原型失败时的处理

若浏览器终端的显示或输入不满足，先保留复现序列并评估修复或替换组件。若 Tailscale Serve 的 WebSocket 握手或长连接失败，回到直接 tailnet HTTP/WS 验证；不启用 Funnel，也不因此放弃跨网要求。

若 tmux 附着/重绘不满足，定位终端尺寸、编码及桥接方式后再决定是否换宿主。若 agent 认证或启动失败，先在同一 Mac 用户环境复现，不能通过关闭 agent 权限保护绕过。

Mac 关机、休眠、Tailscale 下线或 tmux 被终止时，明确展示不可恢复状态，不能显示假在线。

## 完成边界

上述首版核心用例在真实 iPhone Safari 与蜂窝网络通过，才进入“连接、能跑”的交付状态。当前已有运行中的私有 Web 原型，不等于完整首版验收完成。
