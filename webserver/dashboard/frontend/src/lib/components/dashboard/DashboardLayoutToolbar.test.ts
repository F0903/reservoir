import { fireEvent, render, screen } from "@testing-library/svelte";
import { describe, expect, it, vi } from "vitest";
import DashboardLayoutToolbar from "./DashboardLayoutToolbar.svelte";

describe("DashboardLayoutToolbar", () => {
    const noop = () => {};

    it("renders a dashboard refresh action when provided", async () => {
        const onRefresh = vi.fn();
        render(DashboardLayoutToolbar, {
            props: {
                editing: false,
                onEdit: noop,
                onRefresh,
                onReset: noop,
                onSave: noop,
            },
        });

        await fireEvent.click(screen.getByRole("button", { name: "Refresh dashboard metrics" }));

        expect(onRefresh).toHaveBeenCalledOnce();
    });

    it.each([
        { reason: "metrics are loading", editing: false, refreshing: true },
        { reason: "editing the layout", editing: true, refreshing: false },
    ])("disables dashboard refresh while $reason", ({ editing, refreshing }) => {
        render(DashboardLayoutToolbar, {
            props: {
                editing,
                refreshing,
                onEdit: noop,
                onRefresh: noop,
                onReset: noop,
                onSave: noop,
            },
        });

        expect(screen.getByRole("button", { name: "Refresh dashboard metrics" })).toBeDisabled();
    });
});
