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

const sheetLabels = {
  project: ["SESSION CONTEXT", "切换项目"],
  path: ["WORKING DIRECTORY", "切换路径"],
  model: ["AGENT CONTEXT", "Agent 与模型"],
  status: ["LIVE STATUS", "连接与会话状态"],
  actions: ["COMMAND ACTIONS", "指令操作"],
  settings: ["PRIVATE CONSOLE", "设置"],
};

function titleCase(value) {
  return value ? value.charAt(0).toUpperCase() + value.slice(1) : "Agent";
}

function compactPath(value) {
  return value.replace(/^\/Users\/[^/]+(?=\/|$)/, "~");
}

function updateRuntimeContext() {
  const agentName = titleCase(runtimeContext.agent_id);
  const compactDirectory = compactPath(runtimeContext.working_directory || "目录不可用");
  document.querySelector("#project-name").textContent = runtimeContext.workspace_name || "当前目录";
  document.querySelector("#working-directory").textContent = compactDirectory;
  document.querySelector("#working-directory").title = runtimeContext.working_directory;
  document.querySelector("#agent-name").textContent = runtimeContext.agent_id.toUpperCase();
  document.querySelector("#terminal-agent-name").textContent = agentName;
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
}

async function loadRuntimeContext() {
  try {
    const response = await fetch("/api/v1/context", { headers: { Accept: "application/json" } });
    if (!response.ok) throw new Error(`context request returned ${response.status}`);
    Object.assign(runtimeContext, await response.json());
    updateRuntimeContext();
  } catch {
    document.querySelector("#working-directory").textContent = "上下文读取失败";
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

function connect() {
  if (socket && (socket.readyState === WebSocket.OPEN || socket.readyState === WebSocket.CONNECTING)) return;
  attachmentID = "";
  setStatus("正在连接", "connecting");
  const scheme = location.protocol === "https:" ? "wss" : "ws";
  socket = new WebSocket(`${scheme}://${location.host}/api/v1/terminals/prototype/attach`);

  socket.addEventListener("open", () => {
    fitAddon.fit();
    send("terminal.attach", { cols: terminal.cols, rows: terminal.rows });
  });

  socket.addEventListener("message", (event) => {
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
    attachmentID = "";
    setStatus("连接已断开", "disconnected");
  });
  socket.addEventListener("error", () => setStatus("连接错误", "error"));
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

terminal.onData(sendInput);
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
  if (socket && socket.readyState < WebSocket.CLOSING) socket.close();
  window.setTimeout(connect, 80);
});
document.querySelector("#paste-command").addEventListener("click", pasteIntoComposer);
document.querySelector("#focus-terminal").addEventListener("click", () => {
  closeSheet();
  document.querySelector(".terminal-card").scrollIntoView({ behavior: "smooth", block: "start" });
  window.setTimeout(() => terminal.focus(), 220);
});
document.querySelector("#terminal-action").addEventListener("click", () => {
  document.querySelector(".terminal-card").scrollIntoView({ behavior: "smooth", block: "start" });
  terminal.focus();
});

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

updateRuntimeContext();
updateComposerState();
fitAndResize();
loadRuntimeContext();
connect();
