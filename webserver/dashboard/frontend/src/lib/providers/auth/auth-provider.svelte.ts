import { browser } from "$app/environment";
import { goto } from "$app/navigation";
import { resolve } from "$app/paths";
import {
    bootstrapAdmin,
    bootstrapStatus,
    type BootstrapRequest,
    type CreateUserRequest,
    type Credentials,
    type UpdateUserRequest,
    type UserInfo,
    changePassword,
    createUser as createManagedUser,
    deleteUser as deleteManagedUser,
    listUsers as listManagedUsers,
    login,
    logout,
    me,
    updateMe,
    updateUser as updateManagedUser,
} from "$lib/api/auth/auth";
import { log } from "$lib/utils/logger";
import UnauthorizedError from "$lib/api/unauthorized-error";

export class AuthProvider {
    user = $state<UserInfo | null>(null);
    loading = $state(true);

    constructor() {
        if (!browser) return;
        this.checkSession().catch(() => {
            // Error is already handled inside checkSession
        });
    }

    // Attempt to fetch the current user to verify session
    private checkSession = async () => {
        log.debug("Checking for existing user session...");
        this.loading = true;
        try {
            const userInfo = await me();
            this.user = userInfo;
            log.debug("Session verified, logged in as:", userInfo.username);
        } catch (err) {
            if (err instanceof UnauthorizedError) {
                await this.redirectToBootstrapIfRequired();
                return;
            }
            log.error("Failed to check session:", err);
        } finally {
            this.loading = false;
        }
    };

    private redirectToBootstrapIfRequired = async () => {
        const status = await bootstrapStatus();
        if (!status.bootstrap_required) return;
        if (window.location.pathname === "/bootstrap") return;

        await goto(resolve("/bootstrap"), { replaceState: true });
    };

    bootstrap = async (req: BootstrapRequest): Promise<UserInfo> => {
        log.debug("Creating bootstrap admin...");
        const user = await bootstrapAdmin(req);
        this.user = user;
        log.debug("Bootstrap admin created:", user.username);
        return user;
    };

    bootstrapStatus = () => bootstrapStatus();

    login = async (creds: Credentials): Promise<UserInfo> => {
        log.debug("Logging in...");
        const user = await login(creds);
        this.user = user;
        log.debug("Logged in as:", user.username);
        return user;
    };

    logout = async () => {
        log.debug("Logging out...");
        try {
            await logout();
        } finally {
            this.user = null;
            goto(resolve("/login"));
        }
    };

    changePassword = async (currentPassword: string, newPassword: string) => {
        log.debug("Changing password...");
        await changePassword(currentPassword, newPassword);
        this.user = await me();
        log.debug("Password changed successfully");
    };

    updateUsername = async (username: string): Promise<UserInfo> => {
        log.debug("Updating username...");
        const user = await updateMe({ username });
        this.user = user;
        log.debug("Username updated:", user.username);
        return user;
    };

    listUsers = async (): Promise<UserInfo[]> => {
        log.debug("Listing users...");
        return [...(await listManagedUsers())];
    };

    createUser = async (req: CreateUserRequest): Promise<UserInfo> => {
        log.debug("Creating managed user...");
        return createManagedUser(req);
    };

    updateUser = async (id: number, req: UpdateUserRequest): Promise<UserInfo> => {
        log.debug("Updating managed user...");
        const user = await updateManagedUser(id, req);
        if (this.user?.id === user.id) {
            this.user = user;
        }
        return user;
    };

    deleteUser = async (id: number): Promise<void> => {
        log.debug("Deleting managed user...");
        await deleteManagedUser(id);
        if (this.user?.id === id) {
            this.user = null;
            await goto(resolve("/login"));
        }
    };
}
