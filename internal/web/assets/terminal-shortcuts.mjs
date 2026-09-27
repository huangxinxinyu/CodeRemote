// Keep clipboard payloads below the 64 KiB WebSocket frame limit after base64 encoding.
const maxPasteBytes = 40 * 1024;

export function createTerminalShortcuts({ attached, returnToLive, sendInput, paste, readClipboard, message }) {
  function ready() {
    if (attached()) return true;
    message("终端未连接，无法发送输入");
    return false;
  }

  return {
    key(name) {
      const data = { escape: "\x1b", interrupt: "\x03" }[name];
      if (!data || !ready()) return;
      returnToLive();
      if (sendInput(data)) message("");
      else message("发送失败，请检查终端连接");
    },

    async paste() {
      if (!ready()) return;
      if (!readClipboard) {
        message("此页面需要 HTTPS 才能一键读取剪贴板；也可在原生输入栏长按粘贴");
        return;
      }
      let value;
      try {
        value = await readClipboard();
      } catch {
        message("无法读取剪贴板，请在 Safari 提示中允许读取剪贴板");
        return;
      }
      if (!value) {
        message("剪贴板没有可粘贴的文字");
        return;
      }
      if (new TextEncoder().encode(value).length > maxPasteBytes) {
        message("剪贴板文字过长，请分段粘贴");
        return;
      }
      if (!ready()) return;
      returnToLive();
      paste(value);
      message("");
    },
  };
}
