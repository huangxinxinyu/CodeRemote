export function createIOSIMEPunctuationFallback(send) {
  let pending;

  return {
    keydown(event) {
      pending = event.keyCode === 229 && !event.isComposing ? { text: "", delivered: false } : undefined;
    },
    input(event) {
      if (!pending || event.isComposing || event.inputType !== "insertText") return;
      // iOS Chinese keyboards can commit these without an xterm onData event.
      if (event.data === " " || /^\p{P}$/u.test(event.data || "")) pending.text = event.data;
    },
    keyup() {
      if (pending?.text && !pending.delivered) send(pending.text);
      pending = undefined;
    },
    compositionstart() { pending = undefined; },
    compositionend() { pending = undefined; },
    data(value) {
      if (pending) pending.delivered = true;
      send(value);
    },
  };
}
