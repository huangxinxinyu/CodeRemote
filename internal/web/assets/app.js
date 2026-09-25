import { Terminal } from "/assets/xterm.mjs";
import { FitAddon } from "/assets/addon-fit.mjs";

// xterm measures cell width when it opens; wait for the bundled font first.
await document.fonts.load('12.5px "JetBrains Mono"').catch(() => {});

const status = document.querySelector("#connection-status");
const statusLabel = document.querySelector("#connection-label");
const sessionStateLabel = document.querySelector("#session-state-label");
const terminalElement = document.querySelector("#terminal");
const terminalSize = document.querySelector("#terminal-size");
const statusTerminalSize = document.querySelector("#status-terminal-size");
const statusNetwork = document.querySelector("#status-network");
const statusAttachment = document.querySelector("#status-attachment");
const sheet = document.querySelector("#context-sheet");
const sheetTitle = document.querySelector("#sheet-title");
const sheetEyebrow = document.querySelector("#sheet-eyebrow");
const activeSessionName = document.querySelector("#active-session-name");
const sessionList = document.querySelector("#session-list");
const sessionCount = document.querySelector("#session-count");
const historyList = document.querySelector("#history-list");
const historyCount = document.querySelector("#history-count");
const newSessionButton = document.querySelector("#new-session");
const newSessionNav = document.querySelector("#new-session-nav");
const pathForm = document.querySelector("#path-form");
const pathInput = document.querySelector("#path-input");
const pathError = document.querySelector("#path-error");
const directoryList = document.querySelector("#directory-list");
const switchPathButton = document.querySelector("#switch-path");
const openModelPickerButton = document.querySelector("#open-model-picker");
const openSlashMenuButton = document.querySelector("#open-slash-menu");

const runtimeContext = {
  agent_id: "codex",
  working_directory: "",
  workspace_name: "当前目录",
  terminal_id: "prototype",
  model: "native session",
};

const terminal = new Terminal({
  cursorBlink: true,
  fontFamily: '"JetBrains Mono", "PingFang SC", "Hiragino Sans GB", ui-monospace, monospace',
  fontWeight: "400",
  fontWeightBold: "700",
  fontSize: 12.5,
  lineHeight: 1.1,
  minimumContrastRatio: 3,
  scrollback: 5000,
  theme: {
    background: "#282c34",
    foreground: "#ffffff",
    cursor: "#ffffff",
    cursorAccent: "#282c34",
    selectionBackground: "#5a6372aa",
    black: "#1d1f21",
    brightBlack: "#666666",
    green: "#b5bd68",
    brightGreen: "#b9ca4a",
    yellow: "#f0c674",
    brightYellow: "#e7c547",
    red: "#cc6666",
    brightRed: "#d54e53",
    blue: "#81a2be",
    brightBlue: "#7aa6da",
    cyan: "#8abeb7",
    brightCyan: "#70c0b1",
    magenta: "#b294bb",
    brightMagenta: "#c397d8",
    white: "#c5c8c6",
    brightWhite: "#eaeaea",
  },
});
const fitAddon = new FitAddon();
terminal.loadAddon(fitAddon);
terminal.open(terminalElement);
const terminalTextarea = terminalElement.querySelector(".xterm-helper-textarea");

let socket;
let attachmentID = "";
let resizeTimer;
let previousFocus;
let activeTerminalID = runtimeContext.terminal_id;
let connectedTerminalID = "";
let connectionGeneration = 0;
let terminalContexts = [];
let historyThreads = [];
let creatingSession = false;
let resumingThreadID = "";
let deletingTerminalID = "";
let sessionListError = "";
const terminalTouch = { active: false, moved: false, pendingFocus: false, lastY: 0, remainder: 0 };

const sheetLabels = {
  path: ["WORKING DIRECTORY", "切换项目 / 路径"],
  model: ["AGENT CONTEXT", "Agent 与模型"],
  status: ["LIVE STATUS", "连接与会话状态"],
  sessions: ["AGENT SESSIONS", "对话与终端"],
  settings: ["PRIVATE CONSOLE", "设置"],
};

function titleCase(value) {
  return value ? value.charAt(0).toUpperCase() + value.slice(1) : "Agent";
}

function compactPath(value) {
  return value.replace(/^\/Users\/[^/]+(?=\/|$)/, "~");
}

function directoryName(value) {
  return value.replace(/\/+$/, "").split("/").pop() || "/";
}

function rememberedTerminalID() {
  try {
    return localStorage.getItem("code-remote.active-terminal") || "";
  } catch {
    return "";
  }
}

function rememberTerminalID(terminalID) {
  try {
    if (terminalID) {
      localStorage.setItem("code-remote.active-terminal", terminalID);
    } else {
      localStorage.removeItem("code-remote.active-terminal");
    }
  } catch {
    // Private browsing may reject storage; the live session still works.
  }
}

function sessionDisplayName(context) {
	const nativeTitle = context.title?.trim();
	const workspaceName = context.workspace_name?.trim();
	const workspaceSuffix = workspaceName ? ` | ${workspaceName}` : "";
	if (nativeTitle && nativeTitle !== workspaceName) {
		if (workspaceSuffix && nativeTitle.endsWith(workspaceSuffix)) {
			return nativeTitle.slice(0, -workspaceSuffix.length).trim() || nativeTitle;
		}
		return nativeTitle;
	}
	if (context.terminal_id === "prototype") return "初始对话";
	const index = terminalContexts.findIndex((item) => item.terminal_id === context.terminal_id);
	return index >= 0 ? `对话 ${index + 1}` : "新对话";
}

function updateRuntimeContext() {
  const agentName = titleCase(runtimeContext.agent_id);
  const compactDirectory = compactPath(runtimeContext.working_directory || "目录不可用");
  document.querySelector("#project-name").textContent = runtimeContext.workspace_name || "当前目录";
  document.querySelector("#working-directory").textContent = compactDirectory;
  document.querySelector("#working-directory").title = runtimeContext.working_directory;
  document.querySelector("#agent-name").textContent = runtimeContext.agent_id.toUpperCase();
  document.querySelector("#terminal-agent-name").textContent = agentName;
  activeSessionName.textContent = runtimeContext.terminal_id ? sessionDisplayName(runtimeContext) : "暂无终端";
  document.querySelectorAll("[data-current-project]").forEach((element) => {
    element.textContent = runtimeContext.workspace_name || "当前目录";
  });
  document.querySelectorAll("[data-current-path]").forEach((element) => {
    element.textContent = compactDirectory;
    element.title = runtimeContext.working_directory;
  });
  document.querySelectorAll("[data-current-agent]").forEach((element) => {
    element.textContent = agentName;
  });
  document.querySelectorAll("[data-current-terminal]").forEach((element) => {
    element.textContent = runtimeContext.terminal_id || "暂无终端";
  });
  document.querySelectorAll("[data-codex-command]").forEach((element) => {
    element.disabled = runtimeContext.agent_id !== "codex";
  });
}

async function loadRuntimeContext() {
  try {
    const response = await fetch("/api/v1/context", { headers: { Accept: "application/json" } });
    if (!response.ok) throw new Error(`context request returned ${response.status}`);
    Object.assign(runtimeContext, await response.json());
    activeTerminalID = runtimeContext.terminal_id;
    updateRuntimeContext();
  } catch {
    document.querySelector("#working-directory").textContent = "上下文读取失败";
  }
}

function renderSessionList() {
  sessionList.replaceChildren();
  sessionCount.textContent = String(terminalContexts.length);
  if (terminalContexts.length === 0) {
    const empty = document.createElement("p");
    empty.className = "session-list-empty";
    empty.textContent = "还没有可用对话";
    sessionList.append(empty);
    renderSessionListError();
    return;
  }

  terminalContexts.forEach((context) => {
    const row = document.createElement("div");
    row.className = "session-row-shell";
    row.setAttribute("role", "listitem");

    const button = document.createElement("button");
    button.type = "button";
    button.className = "session-row";
    if (context.terminal_id === activeTerminalID) button.classList.add("is-current");

    const icon = document.createElement("span");
    icon.className = "session-row-icon";
    icon.textContent = ">_";

    const copy = document.createElement("span");
    copy.className = "session-row-copy";
    const title = document.createElement("strong");
    title.textContent = sessionDisplayName(context);
    const detail = document.createElement("code");
    detail.textContent = `${context.agent_id.toUpperCase()} · ${compactPath(context.working_directory)}`;
    copy.append(title, detail);

    const state = document.createElement("span");
    state.className = "session-row-state";
    state.textContent = context.terminal_id === activeTerminalID ? "CURRENT" : "打开";

    button.append(icon, copy, state);
    button.addEventListener("click", () => switchTerminal(context.terminal_id));

    const endButton = document.createElement("button");
    endButton.type = "button";
    endButton.className = "session-row-end";
    endButton.textContent = deletingTerminalID === context.terminal_id ? "…" : "结束";
    endButton.disabled = Boolean(deletingTerminalID);
    endButton.setAttribute("aria-label", `结束 ${sessionDisplayName(context)}`);
    endButton.addEventListener("click", () => endTerminal(context.terminal_id));

    row.append(button, endButton);
    sessionList.append(row);
  });
  renderSessionListError();
}

function renderSessionListError() {
  if (sessionListError) {
    const error = document.createElement("p");
    error.className = "session-list-empty";
    error.textContent = sessionListError;
    sessionList.prepend(error);
  }
}

function renderHistoryList() {
  historyList.replaceChildren();
  historyCount.textContent = String(historyThreads.length);
  if (historyThreads.length === 0) {
    const empty = document.createElement("p");
    empty.className = "session-list-empty";
    empty.textContent = "当前目录没有已保存的 Codex 对话";
    historyList.append(empty);
    return;
  }

  historyThreads.forEach((thread) => {
    const button = document.createElement("button");
    button.type = "button";
    button.className = "session-row history-row";
    button.setAttribute("role", "listitem");
    button.disabled = Boolean(resumingThreadID);

    const icon = document.createElement("span");
    icon.className = "session-row-icon";
    icon.textContent = "↻";

    const copy = document.createElement("span");
    copy.className = "session-row-copy";
    const title = document.createElement("strong");
    title.textContent = thread.name?.trim() || thread.preview?.trim() || "未命名对话";
    const detail = document.createElement("code");
    const updated = thread.updated_at ? new Date(thread.updated_at).toLocaleString("zh-CN", { month: "numeric", day: "numeric", hour: "2-digit", minute: "2-digit" }) : "时间未知";
    detail.textContent = thread.preview?.trim() && thread.preview.trim() !== title.textContent ? `${thread.preview.trim()} · ${updated}` : updated;
    copy.append(title, detail);

    const state = document.createElement("span");
    state.className = "session-row-state";
    state.textContent = resumingThreadID === thread.id ? "恢复中" : "恢复";

    button.append(icon, copy, state);
    button.addEventListener("click", () => resumeHistoryThread(thread.id));
    historyList.append(button);
  });
}

async function loadHistory() {
  historyList.replaceChildren();
  const loading = document.createElement("p");
  loading.className = "session-list-empty";
  loading.textContent = "正在读取 Codex 历史…";
  historyList.append(loading);
  try {
    const response = await fetch("/api/v1/codex/threads" + `?working_directory=${encodeURIComponent(runtimeContext.working_directory)}`, { headers: { Accept: "application/json" } });
    if (!response.ok) throw new Error(`Codex history returned ${response.status}`);
    const payload = await response.json();
    historyThreads = Array.isArray(payload.threads) ? payload.threads : [];
    renderHistoryList();
  } catch {
    historyCount.textContent = "—";
    loading.textContent = "读取 Codex 历史失败，请稍后重试";
  }
}

async function loadTerminals() {
  try {
    const response = await fetch("/api/v1/terminals", { headers: { Accept: "application/json" } });
    if (!response.ok) throw new Error(`terminal list returned ${response.status}`);
    const payload = await response.json();
    terminalContexts = Array.isArray(payload.terminals) ? payload.terminals : [];
    if (terminalContexts.length === 0) {
      showNoActiveTerminal();
      return false;
    }
    const remembered = rememberedTerminalID();
    const selected = terminalContexts.find((context) => context.terminal_id === remembered)
      || terminalContexts.find((context) => context.terminal_id === activeTerminalID)
      || terminalContexts[0];
    activeTerminalID = selected.terminal_id;
    Object.assign(runtimeContext, selected);
    renderSessionList();
    updateRuntimeContext();
    return true;
  } catch {
    sessionList.replaceChildren();
    const error = document.createElement("p");
    error.className = "session-list-empty";
    error.textContent = "读取对话失败，请稍后重试";
    sessionList.append(error);
    return false;
  }
}

function showPathError(message = "") {
  pathError.textContent = message;
  pathError.hidden = !message;
}

function updatePathSwitchAction(path) {
  const value = path.trim();
  switchPathButton.disabled = creatingSession || !value;
  switchPathButton.title = value;
  switchPathButton.textContent = creatingSession
    ? "正在创建并切换…"
    : value
      ? `新建并切换到 ${directoryName(value)}`
      : "请输入目标目录";
}

function renderDirectoryListing(listing) {
  pathInput.value = listing.path;
  updatePathSwitchAction(listing.path);
  directoryList.replaceChildren();
  const rows = [];
  if (listing.parent && listing.parent !== listing.path) {
    rows.push({ name: "..", path: listing.parent, detail: "上级目录" });
  }
  for (const entry of listing.directories || []) {
    rows.push({ name: entry.name, path: entry.path, detail: "打开目录" });
  }
  if (rows.length === 0) {
    const empty = document.createElement("p");
    empty.className = "session-list-empty";
    empty.textContent = "没有可浏览的子目录";
    directoryList.append(empty);
    return;
  }
  for (const row of rows) {
    const button = document.createElement("button");
    button.type = "button";
    button.className = "directory-row";
    button.setAttribute("role", "listitem");
    const copy = document.createElement("span");
    const name = document.createElement("strong");
    const detail = document.createElement("code");
    name.textContent = row.name;
    detail.textContent = compactPath(row.path);
    copy.append(name, detail);
    const action = document.createElement("small");
    action.textContent = row.detail;
    button.append(copy, action);
    button.addEventListener("click", () => loadDirectory(row.path));
    directoryList.append(button);
  }
}

async function loadDirectory(path) {
  showPathError();
  directoryList.replaceChildren();
  const loading = document.createElement("p");
  loading.className = "session-list-empty";
  loading.textContent = "正在读取目录…";
  directoryList.append(loading);
  try {
    const response = await fetch(`/api/v1/directories?path=${encodeURIComponent(path)}`, { headers: { Accept: "application/json" } });
    if (!response.ok) throw new Error(`directory browse returned ${response.status}`);
    renderDirectoryListing(await response.json());
  } catch {
    directoryList.replaceChildren();
    showPathError("目录不存在或当前用户无法访问");
  }
}

function setStatus(text, state) {
  statusLabel.textContent = text;
  status.dataset.state = state;
  document.body.dataset.connection = state;
  statusNetwork.textContent = text;

  const labels = {
    connecting: ["ATTACHING", "等待附着"],
    attached: ["ATTACHED", "已附着"],
    disconnected: ["DETACHED", "已断开"],
    error: ["ERROR", "连接错误"],
    exited: ["EXITED", "进程已退出"],
  };
  const [sessionLabel, attachmentLabel] = labels[state] || labels.error;
  sessionStateLabel.textContent = sessionLabel;
  statusAttachment.textContent = attachmentLabel;
}

function bytesToBase64(bytes) {
  let binary = "";
  for (const byte of bytes) binary += String.fromCharCode(byte);
  return btoa(binary);
}

function base64ToBytes(encoded) {
  const binary = atob(encoded);
  const bytes = new Uint8Array(binary.length);
  for (let index = 0; index < binary.length; index += 1) bytes[index] = binary.charCodeAt(index);
  return bytes;
}

function send(type, payload) {
  if (!socket || socket.readyState !== WebSocket.OPEN) return false;
  socket.send(JSON.stringify({ v: 1, type, payload }));
  return true;
}

function sendInput(data) {
  if (!attachmentID) return false;
  return send("terminal.input", {
    attachment_id: attachmentID,
    data_base64: bytesToBase64(new TextEncoder().encode(data)),
  });
}

function tmuxMouseWheelSequence(direction, clientX, clientY) {
  const bounds = terminalElement.getBoundingClientRect();
  const relativeX = Math.max(0, Math.min(bounds.width - 1, clientX - bounds.left));
  const relativeY = Math.max(0, Math.min(bounds.height - 1, clientY - bounds.top));
  const column = Math.max(1, Math.min(terminal.cols, Math.floor((relativeX / bounds.width) * terminal.cols) + 1));
  const row = Math.max(1, Math.min(terminal.rows, Math.floor((relativeY / bounds.height) * terminal.rows) + 1));
  const button = direction < 0 ? 64 : 65;
  return `\x1b[<${button};${column};${row}M`;
}

function installTerminalTouchScrolling() {
  const pixelsPerStep = 22;

  terminalElement.addEventListener("touchstart", (event) => {
    if (event.touches.length !== 1) {
      terminalTouch.active = false;
      return;
    }
    terminalTouch.active = true;
    terminalTouch.moved = false;
    terminalTouch.pendingFocus = false;
    terminalTouch.lastY = event.touches[0].clientY;
    terminalTouch.remainder = 0;
  }, { capture: true, passive: true });

  terminalElement.addEventListener("touchmove", (event) => {
    if (!terminalTouch.active || event.touches.length !== 1 || !attachmentID) return;
    const touch = event.touches[0];
    terminalTouch.remainder += terminalTouch.lastY - touch.clientY;
    terminalTouch.lastY = touch.clientY;
    const steps = Math.min(6, Math.floor(Math.abs(terminalTouch.remainder) / pixelsPerStep));
    if (steps === 0) return;

    event.preventDefault();
    event.stopPropagation();
    terminalTouch.moved = true;
    terminalTouch.pendingFocus = false;
    terminal.blur();
    const direction = terminalTouch.remainder > 0 ? 1 : -1;
    const sequence = tmuxMouseWheelSequence(direction, touch.clientX, touch.clientY);
    for (let index = 0; index < steps; index += 1) sendInput(sequence);
    terminalTouch.remainder -= direction * steps * pixelsPerStep;
  }, { capture: true, passive: false });

  const endGesture = (event) => {
    if (event.type === "touchend" && terminalTouch.pendingFocus && !terminalTouch.moved && attachmentID) {
      send("terminal.focus", { attachment_id: attachmentID });
    }
    terminalTouch.active = false;
    terminalTouch.moved = false;
    terminalTouch.pendingFocus = false;
    terminalTouch.remainder = 0;
  };
  terminalElement.addEventListener("touchend", endGesture, { capture: true, passive: true });
  terminalElement.addEventListener("touchcancel", endGesture, { capture: true, passive: true });
}

function fitAndResize() {
  try {
    fitAddon.fit();
  } catch {
    return;
  }
  const size = `${terminal.cols} × ${terminal.rows}`;
  terminalSize.textContent = size;
  statusTerminalSize.textContent = size;
  if (attachmentID) {
    send("terminal.resize", {
      attachment_id: attachmentID,
      cols: terminal.cols,
      rows: terminal.rows,
    });
  }
}

function connectToTerminal(terminalID = activeTerminalID, force = false) {
  if (!force && connectedTerminalID === terminalID && socket && (socket.readyState === WebSocket.OPEN || socket.readyState === WebSocket.CONNECTING)) return;
  const generation = ++connectionGeneration;
  if (socket && socket.readyState < WebSocket.CLOSING) socket.close();
  attachmentID = "";
  connectedTerminalID = terminalID;
  setStatus("正在连接", "connecting");
  const scheme = location.protocol === "https:" ? "wss" : "ws";
  socket = new WebSocket(`${scheme}://${location.host}/api/v1/terminals/${encodeURIComponent(terminalID)}/attach`);

  socket.addEventListener("open", () => {
    if (generation !== connectionGeneration) return;
    fitAddon.fit();
    send("terminal.attach", { cols: terminal.cols, rows: terminal.rows });
  });

  socket.addEventListener("message", (event) => {
    if (generation !== connectionGeneration) return;
    let message;
    try {
      message = JSON.parse(event.data);
    } catch {
      setStatus("协议错误", "error");
      return;
    }

    if (message.type === "terminal.attached") {
      attachmentID = message.payload.attachment_id;
      setStatus("已连接", "attached");
      fitAndResize();
      if (document.activeElement === terminalTextarea) {
        send("terminal.focus", { attachment_id: attachmentID });
      }
      return;
    }
    if (message.type === "terminal.output" && message.payload.attachment_id === attachmentID) {
      terminal.write(base64ToBytes(message.payload.data_base64));
      return;
    }
    if (message.type === "terminal.exited") {
      attachmentID = "";
      setStatus("进程已退出", "exited");
      return;
    }
    if (message.type === "terminal.error") {
      setStatus(message.payload.message || "终端错误", "error");
    }
  });

  socket.addEventListener("close", () => {
    if (generation !== connectionGeneration) return;
    attachmentID = "";
    setStatus("连接已断开", "disconnected");
  });
  socket.addEventListener("error", () => {
    if (generation === connectionGeneration) setStatus("连接错误", "error");
  });
}

function nativeCommandSequence(command, submit = true) {
  const bracketed = "\x1b[200~" + command + "\x1b[201~";
  return submit ? bracketed + "\r" : bracketed;
}

function sendNativeCommand(command, submit = true) {
  if (runtimeContext.agent_id !== "codex" || !attachmentID) return;
  if (!sendInput(nativeCommandSequence(command, submit))) {
    setStatus("发送失败", "error");
    return;
  }
  closeSheet();
  document.querySelector(".terminal-card").scrollIntoView({ behavior: "smooth", block: "start" });
  window.setTimeout(() => terminal.focus(), 120);
}

function openSheet(name) {
  const label = sheetLabels[name];
  if (!label) return;
  previousFocus = document.activeElement;
  sheetEyebrow.textContent = label[0];
  sheetTitle.textContent = label[1];
  document.querySelectorAll("[data-sheet-view]").forEach((view) => {
    view.hidden = view.dataset.sheetView !== name;
  });
  sheet.hidden = false;
  document.querySelector("#sheet-close").focus();
  if (name === "sessions") {
    loadTerminals();
    loadHistory();
  }
  if (name === "path") {
    pathInput.value = runtimeContext.working_directory;
    loadDirectory(runtimeContext.working_directory);
  }
}

function closeSheet() {
  sheet.hidden = true;
  previousFocus?.focus();
}

function switchTerminal(terminalID) {
  const context = terminalContexts.find((item) => item.terminal_id === terminalID);
  if (!context) return;
  if (terminalID === activeTerminalID && attachmentID) {
    closeSheet();
    return;
  }
  activeTerminalID = terminalID;
  rememberTerminalID(terminalID);
  Object.assign(runtimeContext, context);
  updateRuntimeContext();
  renderSessionList();
  terminal.reset();
  closeSheet();
  connectToTerminal(terminalID, true);
}

function showNoActiveTerminal() {
  connectionGeneration += 1;
  if (socket && socket.readyState < WebSocket.CLOSING) socket.close();
  socket = undefined;
  attachmentID = "";
  connectedTerminalID = "";
  activeTerminalID = "";
  rememberTerminalID("");
  runtimeContext.terminal_id = "";
  runtimeContext.title = "";
  terminal.reset();
  renderSessionList();
  updateRuntimeContext();
  setStatus("暂无终端", "disconnected");
}

async function endTerminal(terminalID) {
  if (deletingTerminalID) return;
  const context = terminalContexts.find((item) => item.terminal_id === terminalID);
  if (!context) return;
  const name = sessionDisplayName(context);
  if (!window.confirm(`结束“${name}”？正在运行的 agent 会停止，终端现场将消失；Codex 原生历史仍会保留。`)) return;

  deletingTerminalID = terminalID;
  sessionListError = "";
  renderSessionList();
  try {
    const response = await fetch(`/api/v1/terminals/${encodeURIComponent(terminalID)}`, {
      method: "DELETE",
      headers: { Accept: "application/json" },
    });
    if (!response.ok) throw new Error(`terminal delete returned ${response.status}`);
    const wasActive = terminalID === activeTerminalID;
    terminalContexts = terminalContexts.filter((item) => item.terminal_id !== terminalID);
    if (!wasActive) {
      renderSessionList();
      return;
    }
    const next = terminalContexts[terminalContexts.length - 1];
    if (next) {
      switchTerminal(next.terminal_id);
    } else {
      showNoActiveTerminal();
    }
  } catch {
    sessionListError = "结束终端失败，请稍后重试";
  } finally {
    deletingTerminalID = "";
    renderSessionList();
  }
}

function setCreatingSession(value) {
  creatingSession = value;
  newSessionButton.disabled = value;
  newSessionNav.disabled = value;
  newSessionButton.lastChild.textContent = value ? " 创建中…" : " 新建对话";
  newSessionNav.querySelector("small").textContent = value ? "创建中" : "新对话";
  updatePathSwitchAction(pathInput.value);
}

async function createSession(workingDirectory = "") {
  if (creatingSession) return;
  setCreatingSession(true);
  showPathError();
  try {
    const headers = { Accept: "application/json" };
    const options = { method: "POST", headers };
    if (workingDirectory) {
      headers["Content-Type"] = "application/json";
      options.body = JSON.stringify({ working_directory: workingDirectory });
    }
    const response = await fetch("/api/v1/terminals", {
      ...options,
    });
    if (!response.ok) throw new Error(`terminal create returned ${response.status}`);
    const created = await response.json();
    terminalContexts.push(created);
    renderSessionList();
    switchTerminal(created.terminal_id);
  } catch {
    if (workingDirectory) {
      showPathError("无法在此目录新建对话，请检查路径后重试");
    } else {
      openSheet("sessions");
      const error = document.createElement("p");
      error.className = "session-list-empty";
      error.textContent = "新建对话失败，请稍后重试";
      sessionList.prepend(error);
    }
  } finally {
    setCreatingSession(false);
  }
}

async function resumeHistoryThread(threadID) {
  if (resumingThreadID) return;
  resumingThreadID = threadID;
  renderHistoryList();
  try {
    const response = await fetch(`/api/v1/codex/threads/${encodeURIComponent(threadID)}/resume`, {
      method: "POST",
      headers: { Accept: "application/json", "Content-Type": "application/json" },
      body: JSON.stringify({ working_directory: runtimeContext.working_directory }),
    });
    if (!response.ok) throw new Error(`resume Codex thread returned ${response.status}`);
    const created = await response.json();
    terminalContexts.push(created);
    renderSessionList();
    switchTerminal(created.terminal_id);
  } catch {
    resumingThreadID = "";
    renderHistoryList();
    const error = document.createElement("p");
    error.className = "session-list-empty";
    error.textContent = "恢复历史对话失败，请稍后重试";
    historyList.prepend(error);
  } finally {
    if (resumingThreadID === threadID) {
      resumingThreadID = "";
      renderHistoryList();
    }
  }
}

terminal.onData(sendInput);
installTerminalTouchScrolling();
new ResizeObserver(() => {
  clearTimeout(resizeTimer);
  resizeTimer = window.setTimeout(fitAndResize, 80);
}).observe(terminalElement);

document.querySelectorAll("[data-sheet]").forEach((button) => {
  button.addEventListener("click", () => openSheet(button.dataset.sheet));
});
document.querySelector("#sheet-close").addEventListener("click", closeSheet);
document.querySelector("#sheet-backdrop").addEventListener("click", closeSheet);
document.querySelector("#reconnect").addEventListener("click", () => {
  closeSheet();
  connectToTerminal(activeTerminalID, true);
});
newSessionButton.addEventListener("click", () => createSession());
newSessionNav.addEventListener("click", () => createSession());
pathForm.addEventListener("submit", (event) => {
  event.preventDefault();
  loadDirectory(pathInput.value);
});
pathInput.addEventListener("input", () => updatePathSwitchAction(pathInput.value));
switchPathButton.addEventListener("click", () => createSession(pathInput.value));
openModelPickerButton.addEventListener("click", () => sendNativeCommand("/model"));
openSlashMenuButton.addEventListener("click", () => sendNativeCommand("/", false));
document.querySelectorAll("[data-native-command]").forEach((button) => {
  button.addEventListener("click", () => sendNativeCommand(button.dataset.nativeCommand));
});
document.addEventListener("keydown", (event) => {
  if (event.key === "Escape" && !sheet.hidden) {
    event.preventDefault();
    closeSheet();
  }
});

function updateKeyboardLayout() {
  if (!window.visualViewport) return;
  const keyboardVisible = window.visualViewport.height < window.innerHeight * 0.78;
  document.body.classList.toggle("keyboard-open", keyboardVisible);
  if (keyboardVisible) {
    document.body.style.setProperty("--visible-viewport-height", `${window.visualViewport.height}px`);
  } else {
    document.body.style.removeProperty("--visible-viewport-height");
  }
  window.setTimeout(fitAndResize, 40);
}

window.visualViewport?.addEventListener("resize", updateKeyboardLayout);
terminalTextarea.addEventListener("focus", () => {
  if (attachmentID) {
    if (terminalTouch.active) {
      terminalTouch.pendingFocus = true;
    } else {
      send("terminal.focus", { attachment_id: attachmentID });
    }
  }
  updateKeyboardLayout();
});
terminalTextarea.addEventListener("blur", () => {
  window.setTimeout(updateKeyboardLayout, 80);
});

async function bootstrap() {
  updateRuntimeContext();
  fitAndResize();
  await loadRuntimeContext();
  const hasTerminals = await loadTerminals();
  if (hasTerminals) connectToTerminal(activeTerminalID);
}

bootstrap();
