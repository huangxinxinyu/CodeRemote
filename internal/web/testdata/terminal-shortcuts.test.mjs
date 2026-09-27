import assert from "node:assert/strict";
import test from "node:test";
import { createTerminalShortcuts } from "../assets/terminal-shortcuts.mjs";

function fixture(overrides = {}) {
  const sent = [];
  const pasted = [];
  const messages = [];
  const shortcuts = createTerminalShortcuts({
    attached: () => true,
    returnToLive: () => sent.push("focus"),
    sendInput: (data) => { sent.push(data); return true; },
    paste: (data) => pasted.push(data),
    readClipboard: async () => "第一行\n第二行",
    message: (value) => messages.push(value),
    ...overrides,
  });
  return { shortcuts, sent, pasted, messages };
}

test("Esc and Ctrl+C send terminal control bytes without submitting Enter", () => {
  const { shortcuts, sent } = fixture();
  shortcuts.key("escape");
  shortcuts.key("interrupt");
  assert.deepEqual(sent, ["focus", "\x1b", "focus", "\x03"]);
});

test("F3 sends the xterm function key sequence for native conversation search", () => {
  const { shortcuts, sent } = fixture();
  shortcuts.key("f3");
  assert.deepEqual(sent, ["focus", "\x1bOR"]);
});

test("Paste reads the clipboard and uses terminal paste semantics without Enter", async () => {
  const { shortcuts, sent, pasted } = fixture();
  await shortcuts.paste();
  assert.deepEqual(sent, ["focus"]);
  assert.deepEqual(pasted, ["第一行\n第二行"]);
});

test("Disconnected shortcuts cannot send keys or read clipboard", async () => {
  let reads = 0;
  const { shortcuts, sent, messages } = fixture({
    attached: () => false,
    readClipboard: async () => { reads += 1; return "text"; },
  });
  shortcuts.key("interrupt");
  await shortcuts.paste();
  assert.deepEqual(sent, []);
  assert.equal(reads, 0);
  assert.equal(messages.at(-1), "终端未连接，无法发送输入");
});

test("Paste reports unavailable clipboard access without sending terminal input", async () => {
  const { shortcuts, sent, pasted, messages } = fixture({ readClipboard: undefined });
  await shortcuts.paste();
  assert.deepEqual(sent, []);
  assert.deepEqual(pasted, []);
  assert.match(messages.at(-1), /HTTPS/);
});

test("Paste reports clipboard permission errors without sending terminal input", async () => {
  const { shortcuts, sent, pasted, messages } = fixture({
    readClipboard: async () => { throw new Error("denied"); },
  });
  await shortcuts.paste();
  assert.deepEqual(sent, []);
  assert.deepEqual(pasted, []);
  assert.match(messages.at(-1), /允许读取剪贴板/);
});

test("Paste refuses an oversized clipboard before writing a WebSocket frame", async () => {
  const { shortcuts, sent, pasted, messages } = fixture({
    readClipboard: async () => "中".repeat(14000),
  });
  await shortcuts.paste();
  assert.deepEqual(sent, []);
  assert.deepEqual(pasted, []);
  assert.match(messages.at(-1), /过长/);
});
