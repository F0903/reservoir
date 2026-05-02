import { describe, it, expect, vi, beforeEach, type Mock } from "vitest";
import { AuthProvider } from "./auth-provider.svelte";
import * as authApi from "$lib/api/auth/auth";
import UnauthorizedError from "$lib/api/unauthorized-error";
import { goto } from "$app/navigation";

// Mock Svelte/Kit environment
vi.mock("$app/environment", () => ({
    browser: true,
}));

// Mock $app/navigation
vi.mock("$app/navigation", () => ({
    goto: vi.fn(),
}));

// Mock $app/paths
vi.mock("$app/paths", () => ({
    resolve: (path: string) => path,
}));

// Mock the API
vi.mock("$lib/api/auth/auth", () => ({
    bootstrapAdmin: vi.fn(),
    bootstrapStatus: vi.fn(),
    login: vi.fn(),
    logout: vi.fn(),
    me: vi.fn(),
    changePassword: vi.fn(),
    updateMe: vi.fn(),
}));

// Mock logger
vi.mock("$lib/utils/logger", () => ({
    log: {
        debug: vi.fn(),
        error: vi.fn(),
    },
}));

describe("AuthProvider", () => {
    let provider: AuthProvider;
    const mockUser: authApi.UserInfo = {
        id: 1,
        username: "testuser",
        is_admin: true,
        password_change_required: false,
        created_at: "2024-01-01T00:00:00Z",
        updated_at: "2024-01-01T00:00:00Z",
    };

    beforeEach(() => {
        vi.clearAllMocks();
        (authApi.me as Mock).mockResolvedValue(mockUser);
        (authApi.bootstrapStatus as Mock).mockResolvedValue({ bootstrap_required: false });
    });

    it("should initialize and check session automatically in browser", async () => {
        provider = new AuthProvider();

        // Wait for constructor's checkSession to finish
        await vi.waitFor(() => expect(provider.loading).toBe(false));

        expect(authApi.me).toHaveBeenCalled();
        expect(provider.user).toEqual(mockUser);
    });

    it("should handle no active session during init", async () => {
        (authApi.me as Mock).mockRejectedValue(new UnauthorizedError());
        provider = new AuthProvider();

        await vi.waitFor(() => expect(provider.loading).toBe(false));

        expect(provider.user).toBeNull();
    });

    it("should redirect to bootstrap when no session exists and bootstrap is required", async () => {
        (authApi.me as Mock).mockRejectedValue(new UnauthorizedError());
        (authApi.bootstrapStatus as Mock).mockResolvedValue({ bootstrap_required: true });
        provider = new AuthProvider();

        await vi.waitFor(() => expect(provider.loading).toBe(false));

        expect(authApi.bootstrapStatus).toHaveBeenCalled();
        expect(goto).toHaveBeenCalledWith("/bootstrap", { replaceState: true });
    });

    it("should create bootstrap admin and store the returned user", async () => {
        (authApi.bootstrapAdmin as Mock).mockResolvedValue(mockUser);
        provider = new AuthProvider();
        await vi.waitFor(() => expect(provider.loading).toBe(false));

        const user = await provider.bootstrap({
            username: "testuser",
            password: "generated-password",
        });

        expect(authApi.bootstrapAdmin).toHaveBeenCalled();
        expect(provider.user).toEqual(mockUser);
        expect(user).toEqual(mockUser);
    });

    it("should handle login correctly", async () => {
        (authApi.login as Mock).mockResolvedValue(mockUser);
        provider = new AuthProvider();
        await vi.waitFor(() => expect(provider.loading).toBe(false));

        const user = await provider.login({ username: "testuser", password: "password" });

        expect(authApi.login).toHaveBeenCalled();
        expect(provider.user).toEqual(mockUser);
        expect(user).toEqual(mockUser);
    });

    it("should handle logout and clear state", async () => {
        provider = new AuthProvider();
        await vi.waitFor(() => expect(provider.loading).toBe(false));

        await provider.logout();

        expect(authApi.logout).toHaveBeenCalled();
        expect(provider.user).toBeNull();
    });

    it("should update username and store the returned user", async () => {
        const updatedUser = { ...mockUser, username: "renamed" };
        (authApi.updateMe as Mock).mockResolvedValue(updatedUser);
        provider = new AuthProvider();
        await vi.waitFor(() => expect(provider.loading).toBe(false));

        const user = await provider.updateUsername("renamed");

        expect(authApi.updateMe).toHaveBeenCalledWith({ username: "renamed" });
        expect(provider.user).toEqual(updatedUser);
        expect(user).toEqual(updatedUser);
    });

    it("should refresh user state after changing password", async () => {
        const refreshedUser = { ...mockUser, updated_at: "2024-01-02T00:00:00Z" };
        (authApi.me as Mock).mockResolvedValueOnce(mockUser).mockResolvedValueOnce(refreshedUser);
        provider = new AuthProvider();
        await vi.waitFor(() => expect(provider.loading).toBe(false));

        await provider.changePassword("old-password", "new-password");

        expect(authApi.changePassword).toHaveBeenCalledWith("old-password", "new-password");
        expect(provider.user).toEqual(refreshedUser);
    });
});
