import { describe, it, expect, vi, beforeEach, type Mock } from "vitest";
import { AuthProvider } from "./auth-provider.svelte";
import * as authApi from "$lib/api/auth/auth";

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
    login: vi.fn(),
    logout: vi.fn(),
    me: vi.fn(),
    changePassword: vi.fn(),
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
        password_change_required: false,
        created_at: "2024-01-01T00:00:00Z",
        updated_at: "2024-01-01T00:00:00Z",
    };

    beforeEach(() => {
        vi.clearAllMocks();
        (authApi.me as Mock).mockResolvedValue(mockUser);
    });

    it("should initialize and check session automatically in browser", async () => {
        provider = new AuthProvider();

        // Wait for constructor's checkSession to finish
        await vi.waitFor(() => expect(provider.loading).toBe(false));

        expect(authApi.me).toHaveBeenCalled();
        expect(provider.user).toEqual(mockUser);
    });

    it("should handle no active session during init", async () => {
        (authApi.me as Mock).mockRejectedValue(new Error("Unauthorized"));
        provider = new AuthProvider();

        await vi.waitFor(() => expect(provider.loading).toBe(false));

        expect(provider.user).toBeNull();
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
});
