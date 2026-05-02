<script lang="ts">
    import ContentShell from "$lib/components/layout/ContentShell.svelte";
    import PagePanel from "$lib/components/layout/PagePanel.svelte";
    import PanelHeading from "$lib/components/layout/PanelHeading.svelte";
    import ErrorBox from "$lib/components/ui/ErrorBox.svelte";
    import PageTitle from "$lib/components/ui/PageTitle.svelte";
    import Button from "$lib/components/ui/input/Button.svelte";
    import Form from "$lib/components/ui/input/Form.svelte";
    import TextInput from "$lib/components/ui/input/TextInput.svelte";
    import Toggle from "$lib/components/ui/input/Toggle.svelte";
    import { getAuthProvider, getToastProvider } from "$lib/context";
    import { log } from "$lib/utils/logger";
    import { KeyRound, RefreshCw, ShieldCheck, ShieldOff, Trash2, UserPlus } from "@lucide/svelte";
    import { onMount } from "svelte";

    const auth = getAuthProvider();
    const toast = getToastProvider();
    type ManagedUser = NonNullable<typeof auth.user>;

    let users = $state<ManagedUser[]>([]);
    let loading = $state(true);
    let error = $state<string | null>(null);
    let busyUserID = $state<number | null>(null);

    let newUsername = $state("");
    let newPassword = $state("");
    let newIsAdmin = $state(false);
    let newRequiresPasswordChange = $state(true);
    let creating = $state(false);
    let createError = $state<string | null>(null);

    let resetTarget = $state<ManagedUser | null>(null);
    let resetPassword = $state("");
    let resetRequiresPasswordChange = $state(true);
    let resetting = $state(false);
    let resetError = $state<string | null>(null);

    const adminCount = $derived(users.filter((user) => user.is_admin).length);

    onMount(() => {
        loadUsers();
    });

    async function loadUsers() {
        loading = true;
        error = null;
        try {
            users = [...(await auth.listUsers())].sort(compareUsers);
        } catch (err) {
            log.error("Failed to load users:", err);
            error = err instanceof Error ? err.message : String(err);
        } finally {
            loading = false;
        }
    }

    async function createManagedUser() {
        if (!newUsername.trim()) {
            createError = "Username must not be empty.";
            return;
        }
        if (!newPassword) {
            createError = "Password must not be empty.";
            return;
        }

        creating = true;
        createError = null;
        try {
            const user = await auth.createUser({
                username: newUsername.trim(),
                password: newPassword,
                is_admin: newIsAdmin,
                password_change_required: newRequiresPasswordChange,
            });
            users = [...users, { ...user }].sort(compareUsers);
            newUsername = "";
            newPassword = "";
            newIsAdmin = false;
            newRequiresPasswordChange = true;
            toast.success("User created.");
        } catch (err) {
            log.error("Failed to create user:", err);
            createError = err instanceof Error ? err.message : String(err);
        } finally {
            creating = false;
        }
    }

    async function updateAdminStatus(user: ManagedUser, isAdmin: boolean) {
        busyUserID = user.id;
        try {
            replaceUser(await auth.updateUser(user.id, { is_admin: isAdmin }));
            toast.success(isAdmin ? "Administrator enabled." : "Administrator disabled.");
        } catch (err) {
            log.error("Failed to update user role:", err);
            toast.error(err instanceof Error ? err.message : String(err));
        } finally {
            busyUserID = null;
        }
    }

    function beginPasswordReset(user: ManagedUser) {
        resetTarget = user;
        resetPassword = "";
        resetRequiresPasswordChange = true;
        resetError = null;
    }

    async function resetUserPassword() {
        if (!resetTarget) {
            return;
        }
        const target = resetTarget;
        if (!resetPassword) {
            resetError = "Password must not be empty.";
            return;
        }

        resetting = true;
        resetError = null;
        try {
            replaceUser(
                await auth.updateUser(target.id, {
                    password: resetPassword,
                    password_change_required: resetRequiresPasswordChange,
                }),
            );
            resetTarget = null;
            resetPassword = "";
            toast.success("Password reset.");
        } catch (err) {
            log.error("Failed to reset password:", err);
            resetError = err instanceof Error ? err.message : String(err);
        } finally {
            resetting = false;
        }
    }

    async function deleteManagedUser(user: ManagedUser) {
        if (!window.confirm(`Delete user "${user.username}"?`)) {
            return;
        }

        busyUserID = user.id;
        try {
            await auth.deleteUser(user.id);
            users = users.filter((item) => item.id !== user.id);
            if (resetTarget?.id === user.id) {
                resetTarget = null;
            }
            toast.success("User deleted.");
        } catch (err) {
            log.error("Failed to delete user:", err);
            toast.error(err instanceof Error ? err.message : String(err));
        } finally {
            busyUserID = null;
        }
    }

    function cancelPasswordReset() {
        resetTarget = null;
    }

    function replaceUser(user: Readonly<ManagedUser>) {
        const nextUser = { ...user };
        users = users.map((item) => (item.id === nextUser.id ? nextUser : item)).sort(compareUsers);
        if (auth.user?.id === nextUser.id) {
            auth.user = nextUser;
        }
        if (resetTarget?.id === nextUser.id) {
            resetTarget = nextUser;
        }
    }

    function compareUsers(a: ManagedUser, b: ManagedUser) {
        return a.username.localeCompare(b.username);
    }

    function adminRemovalDisabled(user: ManagedUser) {
        return user.is_admin && adminCount <= 1;
    }

    function userBusy(user: ManagedUser) {
        return busyUserID === user.id;
    }

    function formatUserDate(value: string) {
        return new Date(value).toLocaleDateString();
    }
</script>

<main class="users-page">
    <ContentShell maxWidth="1100px" gap="1rem">
        <PageTitle --pagetitle-margin-bottom="0">Users</PageTitle>

        <PagePanel mobilePadding="0.9rem">
            <PanelHeading title="Create User">
                {#snippet icon()}
                    <UserPlus size={18} />
                {/snippet}
            </PanelHeading>

            <Form onSubmit={createManagedUser}>
                <div class="create-grid">
                    <TextInput
                        label="Username"
                        bind:value={newUsername}
                        placeholder="operator"
                        maxCharacters={64}
                        disabled={creating}
                    />
                    <TextInput
                        label="Initial Password"
                        bind:value={newPassword}
                        placeholder="Temporary password"
                        censor={true}
                        maxCharacters={128}
                        disabled={creating}
                    />
                    <Toggle label="Administrator" bind:value={newIsAdmin} disabled={creating} />
                    <Toggle
                        label="Require Password Change"
                        bind:value={newRequiresPasswordChange}
                        disabled={creating}
                    />
                </div>

                {#if createError}
                    <ErrorBox>{createError}</ErrorBox>
                {/if}

                <div class="form-actions">
                    <Button type="submit" disabled={creating}>
                        <span class="button-inner">
                            {#if creating}
                                <RefreshCw size={16} class="spin" />
                                Creating...
                            {:else}
                                <UserPlus size={16} />
                                Create User
                            {/if}
                        </span>
                    </Button>
                </div>
            </Form>
        </PagePanel>

        <PagePanel mobilePadding="0.9rem">
            <PanelHeading title="Managed Users">
                {#snippet icon()}
                    <ShieldCheck size={18} />
                {/snippet}
                {#snippet meta()}
                    <span class="count-pill">{users.length}</span>
                {/snippet}
            </PanelHeading>

            {#if error}
                <ErrorBox>{error}</ErrorBox>
            {:else if loading}
                <div class="loading-row">
                    <RefreshCw size={16} class="spin" />
                    Loading users...
                </div>
            {:else}
                <div class="table-wrap">
                    <table>
                        <thead>
                            <tr>
                                <th>User</th>
                                <th>Role</th>
                                <th>Password</th>
                                <th>Created</th>
                                <th aria-label="Actions"></th>
                            </tr>
                        </thead>
                        <tbody>
                            {#each users as user (user.id)}
                                <tr>
                                    <td>
                                        <div class="user-cell">
                                            <strong>{user.username}</strong>
                                            {#if auth.user?.id === user.id}
                                                <span class="self-pill">You</span>
                                            {/if}
                                        </div>
                                    </td>
                                    <td>
                                        <button
                                            type="button"
                                            class="role-button"
                                            class:admin={user.is_admin}
                                            disabled={userBusy(user) || adminRemovalDisabled(user)}
                                            onclick={() => updateAdminStatus(user, !user.is_admin)}
                                        >
                                            {#if user.is_admin}
                                                <ShieldCheck size={14} />
                                                Admin
                                            {:else}
                                                <ShieldOff size={14} />
                                                User
                                            {/if}
                                        </button>
                                    </td>
                                    <td>
                                        <span
                                            class:warning-text={user.password_change_required}
                                            class="status-text"
                                        >
                                            {user.password_change_required
                                                ? "Change required"
                                                : "Current"}
                                        </span>
                                    </td>
                                    <td>{formatUserDate(user.created_at)}</td>
                                    <td>
                                        <div class="row-actions">
                                            <button
                                                type="button"
                                                class="icon-button"
                                                disabled={userBusy(user)}
                                                aria-label={`Reset password for ${user.username}`}
                                                title="Reset password"
                                                onclick={() => beginPasswordReset(user)}
                                            >
                                                <KeyRound size={15} />
                                            </button>
                                            <button
                                                type="button"
                                                class="icon-button danger"
                                                disabled={userBusy(user) ||
                                                    adminRemovalDisabled(user)}
                                                aria-label={`Delete ${user.username}`}
                                                title="Delete user"
                                                onclick={() => deleteManagedUser(user)}
                                            >
                                                <Trash2 size={15} />
                                            </button>
                                        </div>
                                    </td>
                                </tr>
                            {/each}
                        </tbody>
                    </table>
                </div>
            {/if}
        </PagePanel>

        {#if resetTarget}
            {@const target = resetTarget}
            <PagePanel mobilePadding="0.9rem">
                <PanelHeading title="Reset Password">
                    {#snippet icon()}
                        <KeyRound size={18} />
                    {/snippet}
                    {#snippet meta()}
                        <span class="target-pill">{target.username}</span>
                    {/snippet}
                </PanelHeading>

                <Form onSubmit={resetUserPassword}>
                    <div class="reset-grid">
                        <TextInput
                            label="New Password"
                            bind:value={resetPassword}
                            placeholder="Temporary password"
                            censor={true}
                            maxCharacters={128}
                            disabled={resetting}
                        />
                        <Toggle
                            label="Require Password Change"
                            bind:value={resetRequiresPasswordChange}
                            disabled={resetting}
                        />
                    </div>

                    {#if resetError}
                        <ErrorBox>{resetError}</ErrorBox>
                    {/if}

                    <div class="form-actions">
                        <Button
                            type="button"
                            onClick={cancelPasswordReset}
                            disabled={resetting}
                            --btn-background-color="var(--primary-400)"
                        >
                            Cancel
                        </Button>
                        <Button type="submit" disabled={resetting}>
                            <span class="button-inner">
                                {#if resetting}
                                    <RefreshCw size={16} class="spin" />
                                    Resetting...
                                {:else}
                                    <KeyRound size={16} />
                                    Reset Password
                                {/if}
                            </span>
                        </Button>
                    </div>
                </Form>
            </PagePanel>
        {/if}
    </ContentShell>
</main>

<style>
    .users-page {
        display: flex;
        flex-direction: column;
        gap: 1rem;
        min-height: 100%;
    }

    .count-pill,
    .target-pill,
    .self-pill {
        display: inline-flex;
        align-items: center;
        min-height: 1.3rem;
        padding: 0.1rem 0.45rem;
        border-radius: 999px;
        background-color: var(--primary-600);
        color: rgba(255, 255, 255, 0.64);
        font-size: 0.68rem;
        font-weight: 800;
    }

    .create-grid,
    .reset-grid {
        display: grid;
        grid-template-columns: repeat(auto-fit, minmax(min(100%, 220px), 1fr));
        gap: 0.85rem 1rem;
        align-items: end;
    }

    .form-actions {
        display: flex;
        justify-content: flex-end;
        gap: 0.6rem;
        margin-top: 1rem;
    }

    .button-inner,
    .loading-row,
    .role-button,
    .row-actions,
    .user-cell {
        display: flex;
        align-items: center;
    }

    .button-inner,
    .role-button,
    .loading-row {
        gap: 0.45rem;
    }

    .loading-row {
        min-height: 8rem;
        justify-content: center;
        color: rgba(255, 255, 255, 0.58);
        font-weight: 700;
    }

    .table-wrap {
        overflow-x: auto;
    }

    table {
        width: 100%;
        border-collapse: collapse;
        min-width: 760px;
    }

    th,
    td {
        padding: 0.75rem 0.65rem;
        border-bottom: 1px solid rgba(255, 255, 255, 0.06);
        text-align: left;
        vertical-align: middle;
    }

    th {
        color: rgba(255, 255, 255, 0.42);
        font-size: 0.68rem;
        font-weight: 800;
        text-transform: uppercase;
    }

    td {
        color: rgba(255, 255, 255, 0.74);
        font-size: 0.88rem;
    }

    tbody tr:last-child td {
        border-bottom: 0;
    }

    .user-cell {
        gap: 0.5rem;
    }

    .user-cell strong {
        color: var(--secondary-300);
        font-weight: 800;
    }

    .role-button,
    .icon-button {
        border: 1px solid rgba(255, 255, 255, 0.08);
        border-radius: 7px;
        background-color: var(--primary-600);
        color: rgba(255, 255, 255, 0.64);
        transition:
            border-color 120ms ease,
            color 120ms ease,
            background-color 120ms ease;
    }

    .role-button {
        min-width: 5.8rem;
        justify-content: center;
        padding: 0.35rem 0.5rem;
        font-size: 0.75rem;
        font-weight: 800;
    }

    .role-button.admin {
        border-color: color-mix(in srgb, var(--secondary-300) 28%, transparent);
        color: var(--secondary-300);
        background-color: color-mix(in srgb, var(--secondary-800) 22%, var(--primary-600));
    }

    .role-button:hover:enabled,
    .icon-button:hover:enabled {
        border-color: rgba(255, 255, 255, 0.16);
        color: var(--secondary-300);
    }

    .icon-button {
        display: grid;
        place-items: center;
        width: 1.85rem;
        height: 1.85rem;
    }

    .icon-button.danger:hover:enabled {
        border-color: color-mix(in srgb, var(--error-color) 32%, transparent);
        color: var(--error-color);
        background-color: color-mix(in srgb, var(--error-bg) 45%, transparent);
    }

    .role-button:disabled,
    .icon-button:disabled {
        cursor: default;
        opacity: 0.42;
    }

    .row-actions {
        justify-content: flex-end;
        gap: 0.45rem;
    }

    .status-text {
        color: rgba(255, 255, 255, 0.54);
        font-weight: 700;
    }

    .warning-text {
        color: var(--warning-color);
    }

    @keyframes spin {
        from {
            transform: rotate(0deg);
        }
        to {
            transform: rotate(360deg);
        }
    }

    :global(.spin) {
        animation: spin 1s linear infinite;
    }

    @media (max-width: 768px) {
        .form-actions {
            flex-direction: column-reverse;
        }

        .form-actions :global(.btn) {
            width: 100%;
        }
    }
</style>
