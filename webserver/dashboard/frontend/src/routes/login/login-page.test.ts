import { render, screen, waitFor } from "@testing-library/svelte";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { goto } from "$app/navigation";
import LoginPage from "./+page.svelte";

const contextMocks = vi.hoisted(() => ({
    auth: {
        login: vi.fn(),
    },
}));

vi.mock("$app/navigation", () => ({
    goto: vi.fn(),
}));

vi.mock("$app/paths", () => ({
    resolve: (path: string) => path,
}));

vi.mock("#lib/context.ts", () => ({
    getAuthProvider: () => contextMocks.auth,
}));

vi.mock("#lib/utils/logger.ts", () => ({
    log: {
        debug: vi.fn(),
        error: vi.fn(),
    },
}));

describe("login page", () => {
    beforeEach(() => {
        vi.clearAllMocks();
    });

    it.each([
        { id: 1, username: "admin", is_admin: true },
        { id: 2, username: "operator", is_admin: false },
    ])(
        "submits the login form on Enter and redirects to the dashboard (is_admin: $is_admin)",
        async ({ id, username, is_admin }) => {
            contextMocks.auth.login.mockResolvedValue({
                id,
                username,
                is_admin,
                password_change_required: false,
                created_at: "2026-01-01T00:00:00Z",
                updated_at: "2026-01-01T00:00:00Z",
            });
            const user = userEvent.setup();
            render(LoginPage, {
                props: {
                    data: { return: null },
                    params: {},
                },
            });

            await user.type(screen.getByLabelText("Username"), username);
            await user.type(screen.getByLabelText("Password"), "secret{Enter}");

            await waitFor(() =>
                expect(contextMocks.auth.login).toHaveBeenCalledWith({
                    username,
                    password: "secret",
                }),
            );
            await waitFor(() =>
                expect(goto).toHaveBeenCalledWith("/dashboard", {
                    replace: true,
                    refreshAll: true,
                }),
            );
        },
    );
});
