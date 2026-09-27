import assert from "node:assert/strict";
import test from "node:test";
import { terminalViewportLayout } from "../assets/terminal-layout.mjs";

test("iPhone focus keeps the native input visible before the first character", () => {
  const beforeKeyboard = terminalViewportLayout({ ios: true, focused: true, visualHeight: 844, innerHeight: 844 });
  const afterKeyboard = terminalViewportLayout({ ios: true, focused: true, visualHeight: 390, innerHeight: 390 });
  assert.deepEqual(beforeKeyboard, { compact: true, height: 844 });
  assert.deepEqual(afterKeyboard, { compact: true, height: 390 });
});

test("blur restores the normal layout after the iPhone keyboard closes", () => {
  assert.deepEqual(
    terminalViewportLayout({ ios: true, focused: false, visualHeight: 844, innerHeight: 844 }),
    { compact: false, height: 844 },
  );
});

test("other browsers keep the existing viewport-based keyboard detection", () => {
  assert.deepEqual(
    terminalViewportLayout({ ios: false, focused: true, visualHeight: 400, innerHeight: 844 }),
    { compact: true, height: 400 },
  );
  assert.deepEqual(
    terminalViewportLayout({ ios: false, focused: true, visualHeight: 844, innerHeight: 844 }),
    { compact: false, height: 844 },
  );
});
