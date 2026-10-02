export function createIOSIMEPunctuationFallback(send) {
  let pending;

  return {
    keydown(event) {
      // Numeric keyboards may report ordinary key codes instead of 229, even when
      // xterm misses the committed text. Track every non-composition keydown and
      // let onData suppress the fallback when xterm handled it normally.
      pending = !event.isComposing ? { text: "", delivered: false } : undefined;
    },
    input(event) {
      if (!pending || event.isComposing || event.inputType !== "insertText") return;
      // iOS keyboards can commit digits, punctuation, and spaces without an xterm onData event.
      if (event.data === " " || /^[0-9]$/.test(event.data || "") || /^\p{P}$/u.test(event.data || "")) pending.text = event.data;
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
