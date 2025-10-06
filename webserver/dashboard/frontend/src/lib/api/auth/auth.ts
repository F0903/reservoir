import { apiPatch, apiPost } from "../api-methods";

export type UserInfo = {
    id: number;
    username: string;
    password_change_required: boolean;
    created_at: string;
    updated_at: string;
};

export type Credentials = {
    username: string;
    password: string;
};

export function login(creds: Credentials): Promise<Readonly<UserInfo>> {
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
