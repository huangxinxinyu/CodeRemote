import test from "node:test";
import assert from "node:assert/strict";

import {
  createPixelWheelAccumulator,
  momentumLaunchVelocity,
} from "../assets/touch-scroll.mjs";

test("keeps sub-step touch pixels until the configured threshold is reached", () => {
  const wheel = createPixelWheelAccumulator({ pixelsPerStep: 32, maxStepsPerSample: 1 });

  assert.deepEqual(wheel.add(12), { direction: 0, steps: 0 });
  assert.deepEqual(wheel.add(19), { direction: 0, steps: 0 });
  assert.deepEqual(wheel.add(1), { direction: 1, steps: 1 });
  assert.equal(wheel.remainder(), 0);
});

test("emits at most one wheel step per touch sample and preserves extra pixels", () => {
  const wheel = createPixelWheelAccumulator({ pixelsPerStep: 32, maxStepsPerSample: 1 });

  assert.deepEqual(wheel.add(80), { direction: 1, steps: 1 });
  assert.equal(wheel.remainder(), 48);
  assert.deepEqual(wheel.add(0), { direction: 1, steps: 1 });
  assert.equal(wheel.remainder(), 16);
});

test("allows the pixel threshold to be tuned without changing gesture code", () => {
  const precise = createPixelWheelAccumulator({ pixelsPerStep: 40, maxStepsPerSample: 1 });

  assert.deepEqual(precise.add(-39), { direction: 0, steps: 0 });
  assert.deepEqual(precise.add(-1), { direction: -1, steps: 1 });
});

test("reset discards pixels left over from the previous gesture", () => {
  const wheel = createPixelWheelAccumulator({ pixelsPerStep: 32, maxStepsPerSample: 1 });

  wheel.add(24);
  wheel.reset();

  assert.equal(wheel.remainder(), 0);
  assert.deepEqual(wheel.add(8), { direction: 0, steps: 0 });
});

test("reduced-motion preference disables touch momentum", () => {
  const tuning = { scale: 0.28, minimum: 0.32 };

  assert.equal(momentumLaunchVelocity(1.5, tuning, true), 0);
  assert.equal(momentumLaunchVelocity(1.5, tuning, false), 1.5 * tuning.scale);
  assert.equal(momentumLaunchVelocity(1, tuning, false), 0);
});
