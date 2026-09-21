# Web + Tailscale 产品与架构计划

## 目标

完成个人自用的手机 Web 远程终端产品定义，并跑通真实 iPhone Safari、Tailscale、tmux 与 agent 的最小端到端原型。

## 当前阶段

阶段 8 进行中：真实 iPhone 已跑通基础终端链路，正在完成移动端控制台 UI 与剩余真机验收。

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

## 当前产品约束

- iPhone 与 Mac 均连接同一 tailnet；服务不开放普通 LAN 或公网。
- 首版不启用 Tailscale Funnel，不自建公网 Relay。
- agent 沿用 Mac 用户环境与凭据，不修改 HOME/CODEX_HOME，不自动跳过审批。
- tmux 管理终端生命周期；关闭页面只断开附着，不结束 agent。
- Tailscale Serve 的 WebSocket 稳定性必须与直接 tailnet HTTP/WS 一起实测。

## 已知待办

- 新版控制台 UI 需要在真实 iPhone Safari 检查键盘弹起、多行 composer、面板和触控尺寸。
- 目录浏览、多终端列表与持久化恢复尚未实现；当前 project/path/model 面板只提供明确标注的交互壳。
- 需要完成蜂窝网络、中文输入、重连及至少 30 分钟长连接验收。

## 错误记录

- Chrome headless 通过 Serve HTTPS 域名截图时得到 `ERR_CONNECTION_CLOSED`，而 curl HTTPS/WSS 探针及真实 iPhone 均正常；本地视觉检查改走 `127.0.0.1`，不据此修改 Serve 配置。
- 静态检查发现 HTML `data-input` 中的 `\u001b` 不会被浏览器解码为 Esc，而会发送字面文本；先增加页面回归测试，再改为 JavaScript 显式键名映射。
- WebSocket 桥接首次编译时误用了不存在的包级 `websocket.Write`；核对依赖源码后确认应先编码 JSON，再调用连接的 `Write` 方法。
- 检查 npm 发布包时，包含临时文件删除的命令被安全策略拒绝；包下载尚未执行，后续改为保留系统临时目录并由系统回收。
- 首次解析 `tailscale status --json` 时假定 `.Peer` 一定是数组，当前输出为 `null` 导致 jq 迭代失败；已停止推断手机在线状态，改为先检查实际 JSON 结构。
- 递归删除命令被安全策略拒绝；改为逐文件删除，并只对已确认的空目录执行 `rmdir`。
- `make build` 曾把单个 main package 输出到仓库根目录；通过先失败的回归测试定位后，构建目标改为 `BUILD_DIR`。
- 早期验证包装器曾误判 make 的退出码，并遇到 zsh glob 解析差异；后续改为直接验证脚本及使用 bash 运行链接检查。
