import test from "node:test";
import assert from "node:assert/strict";

import { createChromePanelState } from "../assets/chrome-panels.mjs";

function memoryStorage(seed = {}) {
  const values = new Map(Object.entries(seed));
  return {
    getItem(key) { return values.get(key) ?? null; },
    setItem(key, value) { values.set(key, value); },
  };
}

test("top and bottom chrome start expanded", () => {
  const panels = createChromePanelState({ storage: memoryStorage() });

  assert.equal(panels.isCollapsed("top"), false);
  assert.equal(panels.isCollapsed("bottom"), false);
});

test("top and bottom chrome collapse independently and persist", () => {
  const storage = memoryStorage();
  const panels = createChromePanelState({ storage });

  assert.equal(panels.toggle("top"), true);
  assert.equal(panels.isCollapsed("bottom"), false);

  const restored = createChromePanelState({ storage });
  assert.equal(restored.isCollapsed("top"), true);
  assert.equal(restored.isCollapsed("bottom"), false);
});

test("storage failures do not prevent chrome toggles", () => {
  const storage = {
    getItem() { throw new Error("unavailable"); },
    setItem() { throw new Error("unavailable"); },
  };
  const panels = createChromePanelState({ storage });

  assert.equal(panels.toggle("bottom"), true);
  assert.equal(panels.isCollapsed("bottom"), true);
});
