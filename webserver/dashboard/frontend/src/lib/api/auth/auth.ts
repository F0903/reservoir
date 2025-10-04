import { apiPost } from "../api-methods";

export type User = {
    readonly id: number;
    readonly username: string;
    readonly password_hash: string;
    readonly password_reset_required: boolean;
    readonly created_at: string;
    readonly updated_at: string;
};

export type Credentials = {
    username: string;
    password: string;
};

export function login(creds: Credentials): Promise<User> {
    return apiPost<User>("/auth/login", creds, fetch, null);
}

export function logout(): Promise<void> {
    return apiPost<void>("/auth/logout", {}, fetch, null);
}
