import { fireEvent, render, screen, waitFor } from "@testing-library/svelte";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { UserInfo } from "$lib/api/auth/auth";
import UsersPage from "./+page.svelte";

const contextMocks = vi.hoisted(() => ({
    auth: {
        user: null as UserInfo | null,
        listUsers: vi.fn(),
        createUser: vi.fn(),
        updateUser: vi.fn(),
        deleteUser: vi.fn(),
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

const adminUser: UserInfo = {
    id: 1,
    username: "admin",
    is_admin: true,
    password_change_required: false,
    created_at: "2026-05-01T00:00:00Z",
    updated_at: "2026-05-01T00:00:00Z",
};

const regularUser: UserInfo = {
    id: 2,
    username: "operator",
    is_admin: false,
    password_change_required: true,
    created_at: "2026-05-01T00:00:00Z",
    updated_at: "2026-05-01T00:00:00Z",
};

describe("users page", () => {
    beforeEach(() => {
        vi.clearAllMocks();
        contextMocks.auth.user = adminUser;
        contextMocks.auth.listUsers.mockResolvedValue([adminUser, regularUser]);
    });

    it("loads managed users", async () => {
        render(UsersPage);

        await waitFor(() => expect(contextMocks.auth.listUsers).toHaveBeenCalled());

        expect(screen.getByText("admin")).toBeInTheDocument();
        expect(screen.getByText("operator")).toBeInTheDocument();
        expect(screen.getByText("Change required")).toBeInTheDocument();
    });

    it("creates a user", async () => {
        const createdUser = { ...regularUser, id: 3, username: "builder" };
        contextMocks.auth.createUser.mockResolvedValue(createdUser);
        render(UsersPage);
        await waitFor(() => expect(screen.getByText("operator")).toBeInTheDocument());

        await fireEvent.input(screen.getByLabelText("Username"), {
            target: { value: "builder" },
        });
        await fireEvent.input(screen.getByLabelText("Initial Password"), {
            target: { value: "generated-password" },
        });
        await fireEvent.click(screen.getByRole("button", { name: /create user/i }));

        await waitFor(() =>
            expect(contextMocks.auth.createUser).toHaveBeenCalledWith({
                username: "builder",
                password: "generated-password",
                is_admin: false,
                password_change_required: true,
            }),
        );
        expect(contextMocks.toast.success).toHaveBeenCalledWith("User created.");
        expect(screen.getByText("builder")).toBeInTheDocument();
    });

    it("requires a username and password before creating a user", async () => {
        render(UsersPage);
        await waitFor(() => expect(screen.getByText("operator")).toBeInTheDocument());

        const createButton = screen.getByRole("button", { name: /create user/i });
        expect(createButton).toBeDisabled();

        await fireEvent.input(screen.getByLabelText("Username"), {
            target: { value: "builder" },
        });
        expect(createButton).toBeDisabled();

        await fireEvent.input(screen.getByLabelText("Initial Password"), {
            target: { value: "generated-password" },
        });
        expect(createButton).toBeEnabled();
    });

    it("renames the password field when no forced password change is required", async () => {
        render(UsersPage);
        await waitFor(() => expect(screen.getByText("operator")).toBeInTheDocument());

        expect(screen.getByLabelText("Initial Password")).toBeInTheDocument();

        await fireEvent.click(screen.getByLabelText("Require Password Change"));

        expect(screen.getByLabelText("Password")).toBeInTheDocument();
        expect(screen.queryByLabelText("Initial Password")).not.toBeInTheDocument();
    });

    it("promotes a regular user to admin", async () => {
        contextMocks.auth.updateUser.mockResolvedValue({ ...regularUser, is_admin: true });
        render(UsersPage);
        await waitFor(() => expect(screen.getByText("operator")).toBeInTheDocument());

        await fireEvent.click(screen.getByRole("button", { name: /^user$/i }));

        await waitFor(() =>
            expect(contextMocks.auth.updateUser).toHaveBeenCalledWith(regularUser.id, {
                is_admin: true,
            }),
        );
        expect(contextMocks.toast.success).toHaveBeenCalledWith("Administrator enabled.");
    });
});
