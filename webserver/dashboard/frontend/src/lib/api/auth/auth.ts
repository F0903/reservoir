import { apiGet, apiPatch, apiPost, type FetchFn } from "../api-helpers";

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

// Use the AuthProvider instead unless absolutely necessary
export function login(creds: Credentials): Promise<Readonly<UserInfo>> {
    return apiPost<UserInfo>("/auth/login", creds, fetch, null);
}

// Use the AuthProvider instead unless absolutely necessary
export function me(fetchFn: FetchFn = fetch): Promise<Readonly<UserInfo>> {
    return apiGet<UserInfo>("/auth/me", fetchFn, null);
}

// Use the AuthProvider instead unless absolutely necessary
export function logout(): Promise<void> {
    return apiPost<void>("/auth/logout", {}, fetch, null);
}

// Use the AuthProvider instead unless absolutely necessary
export function changePassword(currentPassword: string, newPassword: string): Promise<void> {
    return apiPatch<void>(
        "/auth/change-password",
        { current_password: currentPassword, new_password: newPassword },
        fetch,
    );
}
