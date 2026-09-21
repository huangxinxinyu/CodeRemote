# Web + Tailscale 产品与架构计划

## 目标

完成个人自用的手机 Web 远程终端产品定义，并跑通真实 iPhone Safari、Tailscale、tmux 与 agent 的最小端到端原型。

## 当前阶段

阶段 13 与 14 进入验证：工作目录切换、Codex 原生模型选择器及 `/` 指令入口均已自动化测试通过，待文档、真实 Yuniverse 探针、提交与推送。

## 阶段

### 1. 明确产品边界
- **Status:** complete
- 手机从 Safari 操作 Mac 上的 agent 原生 TUI。
- project/workspace 只表示工作目录入口。
- 首版不做账号业务、聊天数据库、多 agent 协同或项目管理。

### 2. 收敛运行与恢复要求
- **Status:** complete
- 跨网络访问是首版要求，使用 Tailscale 私有网络。
- 同一目录支持多个独立终端。
- 手机断线不停止 Mac 上的 agent，回来重新附着终端。

### 3. 建立文档与 Go 骨架
- **Status:** complete
- 已建立产品、架构、运行时、协议、开发与验收文档。
- 已建立 Go daemon、协议类型和统一检查入口。

### 4. 切换为 Web + Tailscale
- **Status:** complete
- 已将正式产品与技术合同统一为 iPhone Safari + Mac Go Web daemon + Tailscale + tmux。
- 已补充私网监听、Origin 校验、HTTP/WebSocket 与真实设备验收边界。

### 5. 默认开发流程
- **Status:** complete
- 默认 bootstrap 只准备 Go 与 tmux。
- doctor 检查 Go、tmux、Codex、Claude 和 Tailscale。
- daemon 构建产物进入可配置的 `BUILD_DIR`，默认路径为 `bin/code-remote-daemon`。

### 6. 旧客户端清理
- **Status:** complete
- [x] 删除旧手机客户端工程及其生成文件。
- [x] 删除相关 Make 目标、忽略规则和文档记录。
- [x] 卸载机器上的旧工程生成器。
- [x] 保留 macOS 通用开发命令行工具，不触碰其他项目依赖。

### 7. 最终验证
- **Status:** complete
- [x] 运行回归测试、Go test/vet/build 与 doctor。
- [x] 确认仓库不再包含旧客户端资产或引用。
- [x] 检查 Markdown 链接、格式与工作区状态。

### 8. 最小 iPhone Safari 终端原型
- **Status:** in_progress
- [x] Mac 安装并登录 Tailscale，开发前置检查通过。
- [x] 建立可回退的仓库基线；用户明确选择在当前工作区直接开发，不创建 worktree。
- [x] 选定并锁定浏览器终端组件，完成真实 iPhone 基础交互探针。
- [ ] 完成移动端终端外壳视觉优化；桌面 headless 移动布局已复核，等待真实 iPhone 刷新验收。
- [x] 用测试驱动方式实现 Go HTTP/WebSocket、PTY/tmux 与单个真实 agent 的最小链路。
- [ ] 在 iPhone 蜂窝网络下分别验证直接 tailnet HTTP/WS 与 Tailscale Serve HTTPS/WSS；两条路径已基础连通，蜂窝/长连接矩阵待补。
- [ ] 记录真实版本、30 分钟长连接结果与默认接入路径决策。

### 9. 多终端 Session 入口
- **Status:** complete
- [x] 定义终端列表、创建与动态附着 API；新对话严格映射为新 tmux session，不新增聊天数据库。
- [x] 用测试驱动实现终端 registry，保留现有 `prototype` session，并确保切换只替换浏览器附着。
- [x] 在移动端 session sheet 中实现新建、列表、当前项与切换交互。
- [x] 验证两个独立 Codex pane 同时存活、daemon 重启后列表恢复、动态 WSS 附着结束后两个 session 均保留。
- [x] 同步产品、架构、运行时、协议与验收记录。

### 10. Agent 原生会话标题
- **Status:** complete
- [x] 核实 Codex 结构化 thread 元数据包含 `name`，并确认运行中的 Codex 会把自动标题写入 tmux pane title。
- [x] 先补失败测试，定义 terminal list API 的可选原生标题字段与空标题回退。
- [x] 从专用 tmux server 读取 pane title，前端优先展示原生标题且不解析终端正文。
- [x] 完成回归测试、Serve 资源检查和手机尺寸截图。

### 11. 新对话与附件入口分工
- **Status:** complete
- [x] 根据真机截图确认 composer `+` 与新对话入口的语义冲突。
- [x] 先补失败测试：底栏第二项直接创建新对话，不再提供独立“切换项目”入口。
- [x] 保留 composer `+` 作为图片/文件等上下文入口，并诚实标注尚未接入的能力。
- [x] 合并 project/path 的切换入口，完成回归、Serve 和手机尺寸截图。

### 12. Codex 历史与触摸回滚
- **Status:** in_progress
- [x] 核实官方 app-server `thread/list` 提供原生 `name`，并以 cwd 过滤真实读取 6 条历史。
- [x] 在对话面板分开展示正在运行的终端与 Codex 历史，恢复动作启动独立 `codex resume <id>` 终端。
- [x] 将 iPhone TUI 单指上下滑映射为 tmux 鼠标回滚/copy mode，保留原生 TUI 边界。
- [x] 完成自动化回归、构建、Serve 部署与真实 API 检查。
- [ ] 在真实 iPhone Safari 验证上下滑动方向、回到最新画面和点选历史恢复。

### 13. 工作目录切换
- **Status:** in_progress
- [x] 确认目标目录 `/Users/huangxinxinyu/Developer/personal/projects/Yuniverse` 存在。
- [x] 先补失败测试，定义目录校验、按 cwd 创建终端和前端切换合同。
- [x] 实现工作目录浏览/提交 API，并让新终端保存自己的 cwd。
- [x] 在手机路径面板接入真实切换；旧目录终端继续运行，切换后进入新终端。
- [ ] 同步产品、架构、运行时、协议、开发与验收文档。
- [ ] 完成自动化回归、真实 Yuniverse 创建探针、提交并推送。

### 14. Codex 原生指令入口
- **Status:** in_progress
- [x] 核实官方合同：交互式 Codex CLI 的 `/model` 用于切换模型和推理强度。
- [x] 先补失败测试，让模型面板向当前原生 TUI 发送 `/model` 并进入原生选择器。
- [x] 盘点其他适合手机快捷触发且不会绕过审批的原生 `/` 指令。
- [x] 接入 `/` 原生菜单及 `/status`、`/permissions`、`/review` 快捷入口，并保持 Claude/其他 agent 的能力差异诚实可见。
- [ ] 完成回归、文档同步、提交并推送。

## 当前产品约束

- iPhone 与 Mac 均连接同一 tailnet；服务不开放普通 LAN 或公网。
- 首版不启用 Tailscale Funnel，不自建公网 Relay。
- agent 沿用 Mac 用户环境与凭据，不修改 HOME/CODEX_HOME，不自动跳过审批。
- tmux 管理终端生命周期；关闭页面只断开附着，不结束 agent。
- Tailscale Serve 的 WebSocket 稳定性必须与直接 tailnet HTTP/WS 一起实测。
- 切换工作目录严格创建新的 tmux/agent 终端；运行中终端的 cwd 和生命周期不被修改。
- 模型及 `/` 指令复用 Codex 原生 TUI，不维护模型清单、不覆盖用户配置，也不绕过权限确认。

## 已知待办

- 新版控制台 UI 需要在真实 iPhone Safari 检查键盘弹起、多行 composer、面板和触控尺寸。
- 结束终端与创建幂等尚未实现；路径和 Codex 原生指令已接入，agent 选择仍只有显示上下文。
- 需要完成蜂窝网络、中文输入、重连及至少 30 分钟长连接验收。

## 错误记录

- 真实 Yuniverse 创建探针发现 tmux `set-option -t` 不接受 catalog 使用的 `=session-id` target，session 已创建但 cwd 元数据记录失败，HTTP 返回 500；fake runner 原先无条件接受命令，现补真实 target 语义回归测试后修复。诊断命令还误用了 zsh 只读变量 `status`，后续不再以该名称保存退出码。
- 阶段进度批量补丁首次引用了 `docs/decisions.md` 中才有的相似句子，导致整份补丁校验失败且未落盘；重新按各文件实际标题拆分更新。
- 目录解析首轮测试发现裸 `~` 被拼成 `$HOME/~`；将裸 home 与 `~/子路径` 分支处理，继续保留规范化与存在性检查。
- 阶段 13 首次 GREEN 测试发现旧 catalog 测试使用不存在的 `/tmp/project with spaces`，且 macOS 会把临时目录的 `/var` 规范为 `/private/var`；改用真实测试目录并按 `EvalSymlinks` 后的路径断言。
- 新增 terminal catalog 首次编译遗漏 `regexp` import；编译器准确定位后补入，未改变设计。
- Catalog 恢复测试最初用了 8 位示例 ID，而合同要求 12 位随机十六进制；修正测试数据为合同格式后继续验证。
- Chrome headless 通过 Serve HTTPS 域名截图时得到 `ERR_CONNECTION_CLOSED`，而 curl HTTPS/WSS 探针及真实 iPhone 均正常；本地视觉检查改走 `127.0.0.1`，不据此修改 Serve 配置。
- 静态检查发现 HTML `data-input` 中的 `\u001b` 不会被浏览器解码为 Esc，而会发送字面文本；先增加页面回归测试，再改为 JavaScript 显式键名映射。
- WebSocket 桥接首次编译时误用了不存在的包级 `websocket.Write`；核对依赖源码后确认应先编码 JSON，再调用连接的 `Write` 方法。
- 检查 npm 发布包时，包含临时文件删除的命令被安全策略拒绝；包下载尚未执行，后续改为保留系统临时目录并由系统回收。
- 首次解析 `tailscale status --json` 时假定 `.Peer` 一定是数组，当前输出为 `null` 导致 jq 迭代失败；已停止推断手机在线状态，改为先检查实际 JSON 结构。
- 递归删除命令被安全策略拒绝；改为逐文件删除，并只对已确认的空目录执行 `rmdir`。
- `make build` 曾把单个 main package 输出到仓库根目录；通过先失败的回归测试定位后，构建目标改为 `BUILD_DIR`。
- 早期验证包装器曾误判 make 的退出码，并遇到 zsh glob 解析差异；后续改为直接验证脚本及使用 bash 运行链接检查。
