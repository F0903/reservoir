import { vi } from "vitest";

import { describe, it, expect, beforeEach, afterEach } from "vitest";
import { ToastProvider } from "./toast-provider.svelte";

// Mock transitions as they might interfere with testing visibility
vi.mock("svelte/transition", () => ({
    fly: () => ({}),
}));

describe("ToastProvider", () => {
    let provider: ToastProvider;

    beforeEach(() => {
        vi.useFakeTimers();
        provider = new ToastProvider();
    });

    afterEach(() => {
        vi.clearAllTimers();
        vi.useRealTimers();
    });

    it.each([
        ["info", "Hello World"],
        ["success", "Success!"],
        ["error", "Oops"],
    ] as const)("should show a toast of type %s", (type, message) => {
        provider[type](message);
        expect(provider.toasts.length).toBe(1);
        expect(provider.toasts[0].props.message).toBe(message);
        expect(provider.toasts[0].props.type).toBe(type);
    });

    it("should close a toast via handle", () => {
        const handle = provider.info("Test");
        expect(provider.toasts.length).toBe(1);
        handle.close();
        expect(provider.toasts.length).toBe(0);
    });

    it("should handle multiple toasts", () => {
        provider.info("Toast 1");
        provider.info("Toast 2");
        expect(provider.toasts.length).toBe(2);
    });
});
