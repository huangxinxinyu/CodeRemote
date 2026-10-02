import assert from "node:assert/strict";
import test from "node:test";
import { createIOSIMEPunctuationFallback } from "../assets/ios-ime-input.mjs";

function capture() {
  const sent = [];
  return { sent, input: createIOSIMEPunctuationFallback((data) => sent.push(data)) };
}

test("forwards punctuation and spaces that iOS Chinese IME leaves out of xterm onData", () => {
  const { sent, input } = capture();
  for (const character of [" ", "/", "，"]) {
    input.keydown({ keyCode: 229, isComposing: false });
    input.input({ inputType: "insertText", data: character, isComposing: false });
    input.keyup();
  }
  assert.deepEqual(sent, [" ", "/", "，"]);
});

test("does not duplicate input already delivered by xterm", () => {
  const { sent, input } = capture();
  input.keydown({ keyCode: 229, isComposing: false });
  input.input({ inputType: "insertText", data: "/", isComposing: false });
  input.data("/");
  input.keyup();
  assert.deepEqual(sent, ["/"]);
});

test("falls back for ordinary key codes only when xterm misses the committed text", () => {
  const { sent, input } = capture();
  input.keydown({ keyCode: 32, isComposing: false });
  input.input({ inputType: "insertText", data: " ", isComposing: false });
  input.data(" ");
  input.keyup();
  input.keydown({ keyCode: 49, isComposing: false });
  input.input({ inputType: "insertText", data: "1", isComposing: false });
  input.keyup();
  input.keydown({ keyCode: 229, isComposing: false });
  input.compositionstart();
  input.input({ inputType: "insertCompositionText", data: "中", isComposing: true });
  input.compositionend();
  input.keyup();
  input.data("中");
  assert.deepEqual(sent, [" ", "1", "中"]);
});
