// jsdom lacks a few browser APIs that reka-ui's popper/listbox call when a list opens.
Element.prototype.scrollIntoView ??= () => {}
globalThis.ResizeObserver ??= class {
  observe() {}
  unobserve() {}
  disconnect() {}
}
