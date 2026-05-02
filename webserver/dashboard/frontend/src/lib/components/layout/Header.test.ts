import { render, screen } from "@testing-library/svelte";
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

    it("shows the admin badge when the backend has not sent role metadata yet", () => {
        delete (contextMocks.auth.user as { is_admin?: boolean }).is_admin;

        render(Header);

        expect(screen.getByLabelText("Administrator")).toBeInTheDocument();
    });

    it("hides the admin badge for non-admin users", () => {
        contextMocks.auth.user.is_admin = false;

        render(Header);

        expect(screen.queryByLabelText("Administrator")).not.toBeInTheDocument();
    });
});
