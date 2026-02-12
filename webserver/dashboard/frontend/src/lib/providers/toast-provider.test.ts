import { vi } from "vitest";
vi.useFakeTimers();

import { describe, it, expect, beforeEach, afterEach } from "vitest";
import { ToastProvider } from "./toast-provider.svelte";
import { screen, fireEvent, waitFor } from "@testing-library/svelte";

vi.mock("svelte/transition", () => ({
    fly: () => ({}),
}));

describe("ToastProvider", () => {
    let provider: ToastProvider;

    beforeEach(() => {
        provider = new ToastProvider();
        document.body.innerHTML = "";
    });

    afterEach(() => {
        vi.clearAllTimers();
    });

    it("should show an info toast", async () => {
        provider.show({
            type: "info",
            message: "Hello World",
            durationMs: 3000,
        });

        expect(screen.getByText("Hello World")).toBeInTheDocument();
        expect(screen.getByText("OK")).toBeInTheDocument();
    });

    it.skip("should auto-dismiss info toast after duration", async () => {
        provider.show({
            type: "info",
            message: "Auto Dismiss",
            durationMs: 3000,
        });

        expect(screen.getByText("Auto Dismiss")).toBeInTheDocument();

        vi.runAllTimers();
        await vi.advanceTimersByTimeAsync(0);

        expect(screen.queryByText("Auto Dismiss")).not.toBeInTheDocument();
    });

    it("should show an error toast", async () => {
        provider.show({
            type: "error",
            message: "Something went wrong",
            durationMs: 3000,
        });

        expect(screen.getByText("Something went wrong")).toBeInTheDocument();
        expect(screen.getByText("Dismiss")).toBeInTheDocument();
    });

    it("should show an action toast and handle positive action", async () => {
        const onPositive = vi.fn().mockResolvedValue(undefined);

        provider.show({
            type: "action",
            message: "Are you sure?",
            positiveText: "Yes",
            onPositive,
        });

        const yesButton = screen.getByText("Yes");
        await fireEvent.click(yesButton);

        expect(onPositive).toHaveBeenCalled();

        await waitFor(
            () => {
                expect(screen.queryByText("Are you sure?")).not.toBeInTheDocument();
            },
            { timeout: 1000 },
        );
    });
});
