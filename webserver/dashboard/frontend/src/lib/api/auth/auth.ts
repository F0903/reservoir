import { apiPost } from "../api-methods";

export type Credentials = {
    username: string;
    password: string;
};

export function login(creds: Credentials): Promise<void> {
    return apiPost<void>("/auth/login", creds, fetch, null);
}

export function logout(): Promise<void> {
    return apiPost<void>("/auth/logout", {}, fetch, null);
}
