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

vi.mock("$lib/context", () => ({
    getAuthProvider: () => contextMocks.auth,
}));

vi.mock("$lib/utils/logger", () => ({
    log: {
        debug: vi.fn(),
        error: vi.fn(),
    },
}));

describe("login page", () => {
    beforeEach(() => {
        vi.clearAllMocks();
        contextMocks.auth.login.mockResolvedValue({
            id: 1,
            username: "admin",
            is_admin: true,
            password_change_required: false,
            created_at: "2026-01-01T00:00:00Z",
            updated_at: "2026-01-01T00:00:00Z",
        });
    });

    it("submits the login form when pressing Enter in an input", async () => {
        const user = userEvent.setup();
        render(LoginPage, {
            props: {
                data: { return: null },
                params: {},
            },
        });

        await user.type(screen.getByLabelText("Username"), "admin");
        await user.type(screen.getByLabelText("Password"), "secret{Enter}");

        await waitFor(() =>
            expect(contextMocks.auth.login).toHaveBeenCalledWith({
                username: "admin",
                password: "secret",
            }),
        );
        expect(goto).toHaveBeenCalledWith("/dashboard", {
            replaceState: true,
            invalidateAll: true,
        });
    });

    it("redirects non-admin users to the dashboard", async () => {
        contextMocks.auth.login.mockResolvedValue({
            id: 2,
            username: "operator",
            is_admin: false,
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

        await user.type(screen.getByLabelText("Username"), "operator");
        await user.type(screen.getByLabelText("Password"), "secret{Enter}");

        await waitFor(() =>
            expect(goto).toHaveBeenCalledWith("/dashboard", {
                replaceState: true,
                invalidateAll: true,
            }),
        );
    });
});
