export function createPixelWheelAccumulator({ pixelsPerStep, maxStepsPerSample }) {
  if (!Number.isFinite(pixelsPerStep) || pixelsPerStep <= 0) {
    throw new TypeError("pixelsPerStep must be a positive number");
  }
  if (!Number.isInteger(maxStepsPerSample) || maxStepsPerSample <= 0) {
    throw new TypeError("maxStepsPerSample must be a positive integer");
  }

  let remainingPixels = 0;

  return {
    add(pixels) {
      remainingPixels += pixels;
      const availableSteps = Math.floor(Math.abs(remainingPixels) / pixelsPerStep);
      if (availableSteps === 0) return { direction: 0, steps: 0 };

      const direction = remainingPixels > 0 ? 1 : -1;
      const steps = Math.min(availableSteps, maxStepsPerSample);
      remainingPixels -= direction * steps * pixelsPerStep;
      return { direction, steps };
    },
    reset() {
      remainingPixels = 0;
    },
    remainder() {
      return remainingPixels;
    },
  };
}

export function momentumLaunchVelocity(velocity, { scale, minimum }, prefersReducedMotion) {
  if (prefersReducedMotion) return 0;
  const launchVelocity = velocity * scale;
  return Math.abs(launchVelocity) >= minimum ? launchVelocity : 0;
}
