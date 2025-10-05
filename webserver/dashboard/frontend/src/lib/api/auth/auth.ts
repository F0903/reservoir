import { apiPatch, apiPost } from "../api-methods";

export type UserInfo = {
    readonly id: number;
    readonly username: string;
    readonly password_change_required: boolean;
    readonly created_at: string;
    readonly updated_at: string;
};

export type Credentials = {
    readonly username: string;
    readonly password: string;
};

export function login(creds: Credentials): Promise<UserInfo> {
    return apiPost<UserInfo>("/auth/login", creds, fetch, null);
}

export function logout(): Promise<void> {
    return apiPost<void>("/auth/logout", {}, fetch, null);
}

export function changePassword(currentPassword: string, newPassword: string): Promise<void> {
    return apiPatch<void>(
        "/auth/change-password",
        { current_password: currentPassword, new_password: newPassword },
        fetch,
    );
}
