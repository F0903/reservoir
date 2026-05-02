import { browser } from "$app/environment";
import { goto } from "$app/navigation";
import { resolve } from "$app/paths";
import {
    bootstrapAdmin,
    bootstrapStatus,
    type BootstrapRequest,
    type Credentials,
    type UserInfo,
    changePassword,
    login,
    logout,
    me,
    updateMe,
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
}
