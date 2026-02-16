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

    it("should show an info toast", () => {
        provider.info("Hello World");
        expect(provider.toasts.length).toBe(1);
        expect(provider.toasts[0].props.message).toBe("Hello World");
        expect(provider.toasts[0].props.type).toBe("info");
    });

    it("should show a success toast", () => {
        provider.success("Success!");
        expect(provider.toasts.length).toBe(1);
        expect(provider.toasts[0].props.type).toBe("success");
    });

    it("should show an error toast", () => {
        provider.error("Oops");
        expect(provider.toasts.length).toBe(1);
        expect(provider.toasts[0].props.type).toBe("error");
    });

    it("should show an action toast", () => {
        const onPositive = vi.fn();
        provider.action("Confirm?", { onPositive });
        expect(provider.toasts.length).toBe(1);
        expect(provider.toasts[0].props.type).toBe("action");
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
