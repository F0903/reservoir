import "@testing-library/jest-dom";
import { vi } from "vitest";

// Mock transitions
vi.mock("svelte/transition", () => ({
    fly: () => ({}),
    fade: () => ({}),
    slide: () => ({}),
    scale: () => ({}),
    blur: () => ({}),
    draw: () => ({}),
    crossfade: () => [() => ({}), () => ({})],
}));

// JSDOM doesn't support element.animate, which Svelte 5 transitions use.
if (!Element.prototype.animate) {
    Element.prototype.animate = function () {
        return {
            finished: Promise.resolve(),
            cancel: () => {},
            finish: () => {},
            pause: () => {},
            play: () => {},
            reverse: () => {},
            addEventListener: () => {},
            removeEventListener: () => {},
        } as unknown as Animation;
    };
}
