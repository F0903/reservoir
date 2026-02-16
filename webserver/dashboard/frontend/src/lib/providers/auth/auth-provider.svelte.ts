import { browser } from "$app/environment";
import { goto } from "$app/navigation";
import { resolve } from "$app/paths";
import {
    type Credentials,
    type UserInfo,
    changePassword,
    login,
    logout,
    me,
} from "$lib/api/auth/auth";
import { log } from "$lib/utils/logger";

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
        } finally {
            this.loading = false;
        }
    };

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
        log.debug("Password changed successfully");
    };
}
