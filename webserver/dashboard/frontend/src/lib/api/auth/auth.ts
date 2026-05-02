import { apiDelete, apiGet, apiPatch, apiPost, type FetchFn } from "../api-helpers";

export type UserInfo = {
    id: number;
    username: string;
    is_admin: boolean;
    password_change_required: boolean;
    created_at: string;
    updated_at: string;
};

export type Credentials = {
    username: string;
    password: string;
};

export type BootstrapStatus = {
    bootstrap_required: boolean;
};

export type BootstrapRequest = {
    username: string;
    password: string;
};

export type UpdateMeRequest = {
    username: string;
};

export type CreateUserRequest = {
    username: string;
    password: string;
    is_admin: boolean;
    password_change_required?: boolean;
};

export type UpdateUserRequest = {
    username?: string;
    password?: string;
    is_admin?: boolean;
    password_change_required?: boolean;
};

export function bootstrapStatus(fetchFn: FetchFn = fetch): Promise<Readonly<BootstrapStatus>> {
    return apiGet<BootstrapStatus>("/auth/bootstrap", fetchFn, null);
}

export function bootstrapAdmin(
    req: BootstrapRequest,
    fetchFn: FetchFn = fetch,
): Promise<Readonly<UserInfo>> {
    return apiPost<UserInfo>("/auth/bootstrap", req, fetchFn, null);
}

// Use the AuthProvider instead unless absolutely necessary
export function login(creds: Credentials): Promise<Readonly<UserInfo>> {
    return apiPost<UserInfo>("/auth/login", creds, fetch, null);
}

// Use the AuthProvider instead unless absolutely necessary
export function me(fetchFn: FetchFn = fetch): Promise<Readonly<UserInfo>> {
    return apiGet<UserInfo>("/auth/me", fetchFn, null);
}

// Use the AuthProvider instead unless absolutely necessary
export function updateMe(req: UpdateMeRequest): Promise<Readonly<UserInfo>> {
    return apiPatch<UserInfo>("/auth/me", req, fetch);
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

export function listUsers(fetchFn: FetchFn = fetch): Promise<Readonly<UserInfo[]>> {
    return apiGet<UserInfo[]>("/auth/users", fetchFn);
}

export function createUser(req: CreateUserRequest): Promise<Readonly<UserInfo>> {
    return apiPost<UserInfo>("/auth/users", req);
}

export function updateUser(id: number, req: UpdateUserRequest): Promise<Readonly<UserInfo>> {
    return apiPatch<UserInfo>(`/auth/users/${id}`, req);
}

export function deleteUser(id: number): Promise<void> {
    return apiDelete<void>(`/auth/users/${id}`);
}
