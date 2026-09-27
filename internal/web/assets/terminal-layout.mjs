export function terminalViewportLayout({ ios, focused, visualHeight, innerHeight }) {
  const height = Math.min(visualHeight || innerHeight, innerHeight);
  return {
    compact: (ios && focused) || height < innerHeight * 0.78,
    height,
  };
}
