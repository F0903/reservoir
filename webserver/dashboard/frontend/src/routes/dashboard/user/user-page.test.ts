import { fireEvent, render, screen, waitFor } from "@testing-library/svelte";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { UserInfo } from "$lib/api/auth/auth";
import UserPage from "./+page.svelte";

const contextMocks = vi.hoisted(() => ({
    auth: {
        user: null as UserInfo | null,
        loading: false,
        updateUsername: vi.fn(),
        changePassword: vi.fn(),
    },
    toast: {
        success: vi.fn(),
        error: vi.fn(),
    },
}));

vi.mock("$lib/context", () => ({
    getAuthProvider: () => contextMocks.auth,
    getToastProvider: () => contextMocks.toast,
}));

vi.mock("$lib/utils/logger", () => ({
    log: {
        debug: vi.fn(),
        error: vi.fn(),
    },
}));

const currentUser: UserInfo = {
    id: 1,
    username: "admin",
    is_admin: true,
    password_change_required: false,
    created_at: "2026-05-01T00:00:00Z",
    updated_at: "2026-05-01T00:00:00Z",
};

describe("user page", () => {
    beforeEach(() => {
        vi.clearAllMocks();
        contextMocks.auth.user = currentUser;
        contextMocks.auth.loading = false;
    });

    it("updates the current username", async () => {
        const updatedUser = { ...currentUser, username: "operator" };
        contextMocks.auth.updateUsername.mockImplementation(async () => {
            contextMocks.auth.user = updatedUser;
            return updatedUser;
        });
        render(UserPage);

        const username = screen.getByLabelText("Username") as HTMLInputElement;
        await waitFor(() => expect(username.value).toBe("admin"));

        await fireEvent.input(username, { target: { value: "operator" } });
        await fireEvent.click(screen.getByRole("button", { name: /save username/i }));

        await waitFor(() =>
            expect(contextMocks.auth.updateUsername).toHaveBeenCalledWith("operator"),
        );
        expect(contextMocks.toast.success).toHaveBeenCalledWith("Username updated.");
        expect(username.value).toBe("operator");
    });

    it("validates and updates the current password", async () => {
        contextMocks.auth.changePassword.mockResolvedValue(undefined);
        render(UserPage);

        await fireEvent.input(screen.getByLabelText("Current Password"), {
            target: { value: "old-password" },
        });
        await fireEvent.input(screen.getByLabelText("New Password"), {
            target: { value: "new-password" },
        });
        await fireEvent.input(screen.getByLabelText("Confirm Password"), {
            target: { value: "new-password" },
        });
        await fireEvent.click(screen.getByRole("button", { name: /update password/i }));

        await waitFor(() =>
            expect(contextMocks.auth.changePassword).toHaveBeenCalledWith(
                "old-password",
                "new-password",
            ),
        );
        expect(contextMocks.toast.success).toHaveBeenCalledWith("Password updated.");
        expect(screen.getByLabelText("Current Password")).toHaveValue("");
    });

    it("rejects mismatched password confirmation", async () => {
        render(UserPage);

        await fireEvent.input(screen.getByLabelText("Current Password"), {
            target: { value: "old-password" },
        });
        await fireEvent.input(screen.getByLabelText("New Password"), {
            target: { value: "new-password" },
        });
        await fireEvent.input(screen.getByLabelText("Confirm Password"), {
            target: { value: "different-password" },
        });
        await fireEvent.click(screen.getByRole("button", { name: /update password/i }));

        expect(contextMocks.auth.changePassword).not.toHaveBeenCalled();
        expect(screen.getByText("New passwords do not match.")).toBeInTheDocument();
    });
});
