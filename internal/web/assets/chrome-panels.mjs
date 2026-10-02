const panelNames = new Set(["top", "bottom"]);

export function createChromePanelState({ storage, keyPrefix = "code-remote.chrome" }) {
  const collapsed = { top: false, bottom: false };

  for (const panel of panelNames) {
    try {
      collapsed[panel] = storage?.getItem(`${keyPrefix}.${panel}`) === "collapsed";
    } catch {
      collapsed[panel] = false;
    }
  }

  function requirePanel(panel) {
    if (!panelNames.has(panel)) throw new TypeError(`unknown chrome panel: ${panel}`);
  }

  return {
    isCollapsed(panel) {
      requirePanel(panel);
      return collapsed[panel];
    },
    toggle(panel) {
      requirePanel(panel);
      collapsed[panel] = !collapsed[panel];
      try {
        storage?.setItem(`${keyPrefix}.${panel}`, collapsed[panel] ? "collapsed" : "expanded");
      } catch {
        // Private browsing may reject storage; the live layout still works.
      }
      return collapsed[panel];
    },
  };
}
