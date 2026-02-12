import { render, screen, fireEvent } from "@testing-library/svelte";
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import Toast from "./Toast.svelte";
import { ToastHandle } from "../../providers/toast-provider.svelte";

describe("Toast component", () => {
    beforeEach(() => {
        vi.useFakeTimers();
    });

    afterEach(() => {
        vi.useRealTimers();
    });

    it("should render message", () => {
        const handle = new ToastHandle(vi.fn());
        render(Toast, {
            props: {
                type: "info",
                message: "Test Message",
                durationMs: 3000,
                handle,
            },
        });

        expect(screen.getByText("Test Message")).toBeInTheDocument();
    });

    it("should call handle.close after duration", async () => {
        const closer = vi.fn();
        const handle = new ToastHandle(closer);
        render(Toast, {
            props: {
                type: "info",
                message: "Test Message",
                durationMs: 3000,
                handle,
            },
        });

        vi.advanceTimersByTime(3000);
        await vi.advanceTimersByTimeAsync(0);

        expect(closer).toHaveBeenCalled();
    });

    it("should call handle.close when clicking dismiss", async () => {
        const closer = vi.fn();
        const handle = new ToastHandle(closer);
        render(Toast, {
            props: {
                type: "error",
                message: "Error Message",
                durationMs: 3000,
                handle,
            },
        });

        const dismissButton = screen.getByText("Dismiss");
        await fireEvent.click(dismissButton);

        expect(closer).toHaveBeenCalled();
    });
});
