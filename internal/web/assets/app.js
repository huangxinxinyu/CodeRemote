import { Terminal } from "/assets/xterm.mjs";
import { FitAddon } from "/assets/addon-fit.mjs";

const status = document.querySelector("#connection-status");
const statusLabel = document.querySelector("#connection-label");
const sessionStateLabel = document.querySelector("#session-state-label");
const composerConnection = document.querySelector("#composer-connection");
const terminalElement = document.querySelector("#terminal");
const terminalSize = document.querySelector("#terminal-size");
const statusTerminalSize = document.querySelector("#status-terminal-size");
const statusNetwork = document.querySelector("#status-network");
const statusAttachment = document.querySelector("#status-attachment");
const commandInput = document.querySelector("#command-input");
const commandCounter = document.querySelector("#command-counter");
const sendCommandButton = document.querySelector("#send-command");
const sendLabel = document.querySelector("#send-label");
const sentCommands = document.querySelector("#sent-commands");
const sentCommandList = document.querySelector("#sent-command-list");
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
  fontFamily: "SFMono-Regular, Menlo, Monaco, Consolas, monospace",
  fontWeight: "400",
  fontWeightBold: "700",
  fontSize: 12,
  lineHeight: 1.18,
  minimumContrastRatio: 5,
  scrollback: 5000,
  theme: {
    background: "#060c13",
    foreground: "#dce7f5",
    cursor: "#55c8ef",
    cursorAccent: "#060c13",
    selectionBackground: "#284b66aa",
    black: "#05090e",
    brightBlack: "#63748a",
    green: "#3dd6b3",
    brightGreen: "#78ebce",
    yellow: "#efb86b",
    brightYellow: "#ffd398",
    red: "#ff727f",
    brightRed: "#ff9aa3",
    blue: "#69a8ff",
    brightBlue: "#98c2ff",
    cyan: "#55c8ef",
    brightCyan: "#8fddf6",
    magenta: "#aa8cff",
    brightMagenta: "#c9b5ff",
    white: "#dce7f5",
    brightWhite: "#ffffff",
  },
});
const fitAddon = new FitAddon();
terminal.loadAddon(fitAddon);
terminal.open(terminalElement);

let socket;
let attachmentID = "";
let resizeTimer;
let sending = false;
let previousFocus;
let activeTerminalID = runtimeContext.terminal_id;
let connectedTerminalID = "";
let connectionGeneration = 0;
let terminalContexts = [];
let historyThreads = [];
let creatingSession = false;
let resumingThreadID = "";

const sheetLabels = {
  path: ["WORKING DIRECTORY", "切换项目 / 路径"],
  model: ["AGENT CONTEXT", "Agent 与模型"],
  status: ["LIVE STATUS", "连接与会话状态"],
  sessions: ["AGENT SESSIONS", "对话与终端"],
  actions: ["ADD CONTEXT", "添加上下文"],
  settings: ["PRIVATE CONSOLE", "设置"],
};

function titleCase(value) {
  return value ? value.charAt(0).toUpperCase() + value.slice(1) : "Agent";
}

function compactPath(value) {
  return value.replace(/^\/Users\/[^/]+(?=\/|$)/, "~");
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
    localStorage.setItem("code-remote.active-terminal", terminalID);
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
  activeSessionName.textContent = sessionDisplayName(runtimeContext);
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
    element.textContent = runtimeContext.terminal_id || "prototype";
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
    return;
  }

  terminalContexts.forEach((context) => {
    const button = document.createElement("button");
    button.type = "button";
    button.className = "session-row";
    button.setAttribute("role", "listitem");
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
    sessionList.append(button);
  });
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
    const remembered = rememberedTerminalID();
    const selected = terminalContexts.find((context) => context.terminal_id === remembered);
    if (selected) {
      activeTerminalID = selected.terminal_id;
      Object.assign(runtimeContext, selected);
    }
    renderSessionList();
    updateRuntimeContext();
  } catch {
    sessionList.replaceChildren();
    const error = document.createElement("p");
    error.className = "session-list-empty";
    error.textContent = "读取对话失败，请稍后重试";
    sessionList.append(error);
  }
}

function showPathError(message = "") {
  pathError.textContent = message;
  pathError.hidden = !message;
}

function renderDirectoryListing(listing) {
  pathInput.value = listing.path;
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
    connecting: ["ATTACHING", "终端连接中", "等待附着"],
    attached: ["ATTACHED", "可发送指令", "已附着"],
    disconnected: ["DETACHED", "连接已断开", "已断开"],
    error: ["ERROR", "连接异常", "连接错误"],
    exited: ["EXITED", "进程已退出", "进程已退出"],
  };
  const [sessionLabel, composerLabel, attachmentLabel] = labels[state] || labels.error;
  sessionStateLabel.textContent = sessionLabel;
  composerConnection.textContent = composerLabel;
  statusAttachment.textContent = attachmentLabel;
  updateComposerState();
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
  const gesture = { active: false, lastY: 0, remainder: 0 };
  const pixelsPerStep = 22;

  terminalElement.addEventListener("touchstart", (event) => {
    if (event.touches.length !== 1) {
      gesture.active = false;
      return;
    }
    gesture.active = true;
    gesture.lastY = event.touches[0].clientY;
    gesture.remainder = 0;
  }, { capture: true, passive: true });

  terminalElement.addEventListener("touchmove", (event) => {
    if (!gesture.active || event.touches.length !== 1 || !attachmentID) return;
    const touch = event.touches[0];
    gesture.remainder += gesture.lastY - touch.clientY;
    gesture.lastY = touch.clientY;
    const steps = Math.min(6, Math.floor(Math.abs(gesture.remainder) / pixelsPerStep));
    if (steps === 0) return;

    event.preventDefault();
    event.stopPropagation();
    const direction = gesture.remainder > 0 ? 1 : -1;
    const sequence = tmuxMouseWheelSequence(direction, touch.clientX, touch.clientY);
    for (let index = 0; index < steps; index += 1) sendInput(sequence);
    gesture.remainder -= direction * steps * pixelsPerStep;
  }, { capture: true, passive: false });

  const endGesture = () => {
    gesture.active = false;
    gesture.remainder = 0;
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

function resizeComposer() {
  commandInput.style.height = "auto";
  commandInput.style.height = `${Math.min(commandInput.scrollHeight, 108)}px`;
}

function updateComposerState() {
  const length = commandInput.value.length;
  const ready = Boolean(attachmentID) && length > 0 && commandInput.value.trim().length > 0 && !sending;
  commandCounter.textContent = `${length}/2000`;
  sendCommandButton.disabled = !ready;
  sendCommandButton.dataset.state = sending ? "sending" : ready ? "enabled" : "disabled";
  sendLabel.textContent = sending ? "发送中" : "运行";
}

function addSentCommand(command) {
  const item = document.createElement("li");
  const content = document.createElement("span");
  const timestamp = document.createElement("time");
  content.textContent = command;
  timestamp.textContent = new Intl.DateTimeFormat("zh-CN", {
    hour: "2-digit",
    minute: "2-digit",
    hour12: false,
  }).format(new Date());
  item.append(content, timestamp);
  sentCommandList.append(item);
  while (sentCommandList.children.length > 3) sentCommandList.firstElementChild.remove();
  sentCommands.hidden = false;
}

function submitCommand() {
  const command = commandInput.value.replace(/\s+$/, "");
  if (!command.trim() || !attachmentID || sending) return;
  sending = true;
  updateComposerState();
  if (!sendInput(`${command}\r`)) {
    sending = false;
    setStatus("发送失败", "error");
    return;
  }
  addSentCommand(command);
  commandInput.value = "";
  resizeComposer();
  commandInput.blur();
  window.setTimeout(() => {
    document.querySelector(".terminal-card").scrollIntoView({ behavior: "smooth", block: "start" });
  }, 100);
  window.setTimeout(() => {
    sending = false;
    updateComposerState();
  }, 360);
}

function sendNativeCommand(command, submit = true) {
  if (runtimeContext.agent_id !== "codex" || !attachmentID) return;
  if (!sendInput(`${command}${submit ? "\r" : ""}`)) {
    setStatus("发送失败", "error");
    return;
  }
  addSentCommand(command);
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

async function pasteIntoComposer() {
  try {
    const pasted = await navigator.clipboard.readText();
    const start = commandInput.selectionStart;
    const end = commandInput.selectionEnd;
    commandInput.setRangeText(pasted, start, end, "end");
    resizeComposer();
    updateComposerState();
    closeSheet();
    commandInput.focus();
  } catch {
    closeSheet();
    commandInput.focus();
  }
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
  sentCommandList.replaceChildren();
  sentCommands.hidden = true;
  terminal.reset();
  closeSheet();
  connectToTerminal(terminalID, true);
}

function setCreatingSession(value) {
  creatingSession = value;
  newSessionButton.disabled = value;
  newSessionNav.disabled = value;
  newSessionButton.lastChild.textContent = value ? " 创建中…" : " 新建对话";
  newSessionNav.querySelector("small").textContent = value ? "创建中" : "新对话";
  switchPathButton.disabled = value;
  switchPathButton.textContent = value ? "正在创建…" : "在此目录新建对话";
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

commandInput.addEventListener("input", () => {
  resizeComposer();
  updateComposerState();
});
commandInput.addEventListener("keydown", (event) => {
  if (event.key === "Enter" && (event.metaKey || event.ctrlKey)) {
    event.preventDefault();
    submitCommand();
  }
});
sendCommandButton.addEventListener("click", submitCommand);

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
switchPathButton.addEventListener("click", () => createSession(pathInput.value));
openModelPickerButton.addEventListener("click", () => sendNativeCommand("/model"));
openSlashMenuButton.addEventListener("click", () => sendNativeCommand("/", false));
document.querySelectorAll("[data-native-command]").forEach((button) => {
  button.addEventListener("click", () => sendNativeCommand(button.dataset.nativeCommand));
});
document.querySelector("#paste-command").addEventListener("click", pasteIntoComposer);
document.addEventListener("keydown", (event) => {
  if (event.key === "Escape" && !sheet.hidden) {
    event.preventDefault();
    closeSheet();
  }
});

function updateKeyboardLayout() {
  if (!window.visualViewport) return;
  const composerFocused = document.activeElement === commandInput;
  const keyboardVisible = window.visualViewport.height < window.innerHeight * 0.78;
  document.body.classList.toggle("keyboard-open", composerFocused && keyboardVisible);
  window.setTimeout(fitAndResize, 40);
}

window.visualViewport?.addEventListener("resize", updateKeyboardLayout);
commandInput.addEventListener("focus", updateKeyboardLayout);
commandInput.addEventListener("blur", () => {
  window.setTimeout(updateKeyboardLayout, 80);
});

async function bootstrap() {
  updateRuntimeContext();
  updateComposerState();
  fitAndResize();
  await loadRuntimeContext();
  await loadTerminals();
  connectToTerminal(activeTerminalID);
}

bootstrap();
