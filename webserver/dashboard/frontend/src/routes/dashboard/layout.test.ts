import { render, screen, waitFor } from "@testing-library/svelte";
import { goto } from "$app/navigation";
import { createRawSnippet } from "svelte";
import { beforeEach, describe, expect, it, vi } from "vitest";
import DashboardLayout from "./+layout.svelte";

const contextMocks = vi.hoisted(() => ({
    auth: {
        loading: false,
        user: {
            id: 1,
            username: "operator",
            is_admin: false,
            password_change_required: false,
            created_at: "2026-05-01T00:00:00Z",
            updated_at: "2026-05-01T00:00:00Z",
        },
    },
    page: {
        url: new URL("http://localhost/dashboard"),
    },
}));

vi.mock("$app/environment", () => ({
    browser: true,
}));

vi.mock("$app/navigation", () => ({
    goto: vi.fn(),
}));

vi.mock("$app/paths", () => ({
    resolve: (path: string) => path,
}));

vi.mock("$app/state", () => ({
    page: contextMocks.page,
}));

vi.mock("$lib/api/objects/version/version", () => ({
    version: vi.fn().mockResolvedValue({ version: "test-version" }),
}));

vi.mock("$lib/context", () => ({
    getAuthProvider: () => contextMocks.auth,
}));

const children = createRawSnippet(() => ({
    render: () => "<div>Dashboard content</div>",
}));

describe("dashboard layout", () => {
    beforeEach(() => {
        vi.clearAllMocks();
        contextMocks.auth.loading = false;
        contextMocks.auth.user = {
            id: 1,
            username: "operator",
            is_admin: false,
            password_change_required: false,
            created_at: "2026-05-01T00:00:00Z",
            updated_at: "2026-05-01T00:00:00Z",
        };
        contextMocks.page.url = new URL("http://localhost/dashboard");
    });

    it("hides admin-only navigation from non-admin users", () => {
        render(DashboardLayout, { props: { children } });

        expect(screen.getByText("Dashboard")).toBeInTheDocument();
        expect(screen.getByText("Log")).toBeInTheDocument();
        expect(screen.queryByText("Settings")).not.toBeInTheDocument();
        expect(screen.queryByText("Users")).not.toBeInTheDocument();
    });

    it("shows admin-only navigation to admin users", () => {
        contextMocks.auth.user.is_admin = true;

        render(DashboardLayout, { props: { children } });

        expect(screen.getByText("Settings")).toBeInTheDocument();
        expect(screen.getByText("Users")).toBeInTheDocument();
    });

    it("redirects non-admin users away from admin-only routes", async () => {
        contextMocks.page.url = new URL("http://localhost/dashboard/settings");

        render(DashboardLayout, { props: { children } });

        await waitFor(() =>
            expect(goto).toHaveBeenCalledWith("/dashboard", { replaceState: true }),
        );
    });
});
