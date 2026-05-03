import { fireEvent, render, screen, waitFor } from "@testing-library/svelte";
import { goto } from "$app/navigation";
import { beforeEach, describe, expect, it, vi } from "vitest";
import Header from "./Header.svelte";

const contextMocks = vi.hoisted(() => ({
    auth: {
        user: {
            id: 1,
            username: "admin",
            is_admin: true,
            password_change_required: false,
            created_at: "2024-01-01T00:00:00Z",
            updated_at: "2024-01-01T00:00:00Z",
        },
        logout: vi.fn(),
    },
}));

vi.mock("$lib/context", () => ({
    getAuthProvider: () => contextMocks.auth,
}));

vi.mock("$app/navigation", () => ({
    goto: vi.fn(),
}));

vi.mock("$app/paths", () => ({
    resolve: (path: string) => path,
}));

vi.mock("$lib/api/objects/version/version", () => ({
    version: vi.fn().mockResolvedValue({ version: "test-version" }),
}));

describe("Header", () => {
    beforeEach(() => {
        vi.clearAllMocks();
    });

    it("shows an admin badge for admin users", () => {
        contextMocks.auth.user.is_admin = true;

        render(Header);

        expect(screen.getByLabelText("Administrator")).toBeInTheDocument();
    });

    it("hides the admin badge for non-admin users", () => {
        contextMocks.auth.user.is_admin = false;

        render(Header);

        expect(screen.queryByLabelText("Administrator")).not.toBeInTheDocument();
    });

    it("shows the menu toggle for non-admin users when navigation is available", () => {
        contextMocks.auth.user.is_admin = false;

        render(Header, { props: { onToggleMenu: vi.fn() } });

        expect(screen.getByLabelText("Toggle menu")).toBeInTheDocument();
    });

    it("opens the user page from the header user button", async () => {
        render(Header);

        await fireEvent.click(screen.getByRole("button", { name: /open user profile/i }));

        await waitFor(() => expect(goto).toHaveBeenCalledWith("/dashboard/user"));
    });
});
