export function userIsAdmin(user: { is_admin?: boolean } | null | undefined) {
    return user?.is_admin === true;
}
