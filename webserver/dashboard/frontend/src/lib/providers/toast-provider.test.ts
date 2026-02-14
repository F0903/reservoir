import { vi } from "vitest";
import { describe, it, expect, beforeEach, afterEach } from "vitest";
import { ToastProvider } from "./toast-provider.svelte";
import { screen, waitFor } from "@testing-library/svelte";
import userEvent from "@testing-library/user-event";

vi.mock("svelte/transition", () => ({
    fly: () => ({}),
}));

describe("ToastProvider", () => {
    let provider: ToastProvider;
    let user: ReturnType<typeof userEvent.setup>;
    let container: HTMLDivElement;

    beforeEach(() => {
        // Create a specific container for the provider to make cleanup easier and more reliable
        container = document.createElement("div");
        container.id = "toast-container";
        document.body.appendChild(container);

        provider = new ToastProvider(container);
        user = userEvent.setup();
    });

    afterEach(() => {
        document.body.innerHTML = "";
        vi.useRealTimers();
    });

    it("should show an info toast with correct accessibility role", async () => {
        provider.show({
            type: "info",
            message: "Hello World",
            durationMs: 3000,
        });

        const toast = screen.getByRole("status");
        expect(toast).toBeInTheDocument();
        expect(toast).toHaveTextContent("Hello World");
        expect(screen.getByText("OK")).toBeInTheDocument();
    });

    it("should auto-dismiss info toast after duration", async () => {
        provider.show({
            type: "info",
            message: "Auto Dismiss",
            durationMs: 100,
        });

        expect(screen.getByText("Auto Dismiss")).toBeInTheDocument();

        await waitFor(
            () => {
                expect(screen.queryByText("Auto Dismiss")).not.toBeInTheDocument();
            },
            { timeout: 1000 },
        );
    });

    it("should show an error toast with alert role", async () => {
        provider.show({
            type: "error",
            message: "Something went wrong",
            durationMs: 3000,
        });

        const toast = screen.getByRole("alert");
        expect(toast).toBeInTheDocument();
        expect(toast).toHaveTextContent("Something went wrong");
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

        expect(screen.getByRole("status")).toBeInTheDocument();

        const yesButton = screen.getByText("Yes");
        await user.click(yesButton);

        expect(onPositive).toHaveBeenCalled();

        await waitFor(
            () => {
                expect(screen.queryByText("Are you sure?")).not.toBeInTheDocument();
            },
            { timeout: 1000 },
        );
    });

    it("should show multiple toasts and they should coexist", async () => {
        provider.show({
            type: "info",
            message: "First Toast",
            durationMs: 3000,
        });
        provider.show({
            type: "info",
            message: "Second Toast",
            durationMs: 3000,
        });

        expect(screen.getByText("First Toast")).toBeInTheDocument();
        expect(screen.getByText("Second Toast")).toBeInTheDocument();
    });

    it("should allow manual dismissal via button", async () => {
        provider.show({
            type: "info",
            message: "Manual Dismiss",
            durationMs: 10000,
        });

        const okButton = screen.getByText("OK");
        await user.click(okButton);

        await waitFor(() => {
            expect(screen.queryByText("Manual Dismiss")).not.toBeInTheDocument();
        });
    });

    it("should handle negative action in action toast", async () => {
        const onNegative = vi.fn().mockResolvedValue(undefined);

        provider.show({
            type: "action",
            message: "Negative Action Test",
            negativeText: "Cancel",
            onNegative,
        });

        const cancelButton = screen.getByText("Cancel");
        await user.click(cancelButton);

        expect(onNegative).toHaveBeenCalled();

        await waitFor(() => {
            expect(screen.queryByText("Negative Action Test")).not.toBeInTheDocument();
        });
    });

    it("should close toast when handle.close() is called", async () => {
        const handle = provider.show({
            type: "info",
            message: "Handle Close Test",
            durationMs: 10000,
        });

        expect(screen.getByText("Handle Close Test")).toBeInTheDocument();

        handle.close();

        await waitFor(() => {
            expect(screen.queryByText("Handle Close Test")).not.toBeInTheDocument();
        });
    });

    it("should not close toast if action handler fails", async () => {
        // Mock console.error to avoid noisy output during expected error
        const consoleSpy = vi.spyOn(console, "error").mockImplementation(() => {});
        const onPositive = vi.fn().mockRejectedValue(new Error("Action Failed"));

        provider.show({
            type: "action",
            message: "Failing Action",
            onPositive,
        });

        const yesButton = screen.getByText("Yes");
        await user.click(yesButton);

        expect(onPositive).toHaveBeenCalled();

        // Wait a bit to ensure it doesn't close
        await new Promise((resolve) => setTimeout(resolve, 50));
        expect(screen.getByText("Failing Action")).toBeInTheDocument();

        consoleSpy.mockRestore();
    });

    it("should handle multiple toasts with different auto-dismiss durations", async () => {
        provider.show({
            type: "info",
            message: "Short Toast",
            durationMs: 100,
        });
        provider.show({
            type: "info",
            message: "Long Toast",
            durationMs: 500,
        });

        expect(screen.getByText("Short Toast")).toBeInTheDocument();
        expect(screen.getByText("Long Toast")).toBeInTheDocument();

        // Short one should disappear first
        await waitFor(() => {
            expect(screen.queryByText("Short Toast")).not.toBeInTheDocument();
        });
        expect(screen.getByText("Long Toast")).toBeInTheDocument();

        // Then long one
        await waitFor(() => {
            expect(screen.queryByText("Long Toast")).not.toBeInTheDocument();
        });
    });
});
