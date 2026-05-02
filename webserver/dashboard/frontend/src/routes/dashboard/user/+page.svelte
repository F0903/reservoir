<script lang="ts">
    import ErrorBox from "$lib/components/ui/ErrorBox.svelte";
    import PageTitle from "$lib/components/ui/PageTitle.svelte";
    import Button from "$lib/components/ui/input/Button.svelte";
    import Form from "$lib/components/ui/input/Form.svelte";
    import TextInput from "$lib/components/ui/input/TextInput.svelte";
    import { getAuthProvider, getToastProvider } from "$lib/context";
    import { log } from "$lib/utils/logger";
    import { KeyRound, RefreshCw, Save, UserRound } from "@lucide/svelte";

    const auth = getAuthProvider();
    const toast = getToastProvider();

    let username = $state("");
    let savingUsername = $state(false);
    let usernameError = $state<string | null>(null);

    let currentPassword = $state("");
    let newPassword = $state("");
    let confirmPassword = $state("");
    let changingPassword = $state(false);
    let passwordError = $state<string | null>(null);

    let loadedUsername = $state("");

    const normalizedUsername = $derived(username.trim());
    const usernameChanged = $derived(
        !!auth.user && normalizedUsername !== "" && normalizedUsername !== auth.user.username,
    );
    const passwordFormFilled = $derived(
        currentPassword !== "" || newPassword !== "" || confirmPassword !== "",
    );

    $effect(() => {
        const current = auth.user?.username ?? "";
        if (!current || current === loadedUsername) {
            return;
        }

        if (username === "" || username === loadedUsername) {
            username = current;
            loadedUsername = current;
        }
    });

    async function saveUsername() {
        if (!normalizedUsername) {
            usernameError = "Username must not be empty.";
            return;
        }
        if (!usernameChanged) {
            usernameError = null;
            return;
        }

        savingUsername = true;
        usernameError = null;
        try {
            const user = await auth.updateUsername(normalizedUsername);
            username = user.username;
            loadedUsername = user.username;
            toast.success("Username updated.");
        } catch (err) {
            log.error("Failed to update username:", err);
            usernameError = err instanceof Error ? err.message : String(err);
        } finally {
            savingUsername = false;
        }
    }

    async function savePassword() {
        if (!currentPassword || !newPassword || !confirmPassword) {
            passwordError = "All password fields are required.";
            return;
        }
        if (newPassword !== confirmPassword) {
            passwordError = "New passwords do not match.";
            return;
        }
        if (currentPassword === newPassword) {
            passwordError = "New password must be different from current password.";
            return;
        }

        changingPassword = true;
        passwordError = null;
        try {
            await auth.changePassword(currentPassword, newPassword);
            currentPassword = "";
            newPassword = "";
            confirmPassword = "";
            toast.success("Password updated.");
        } catch (err) {
            log.error("Failed to update password:", err);
            passwordError = err instanceof Error ? err.message : String(err);
        } finally {
            changingPassword = false;
        }
    }
</script>

<main class="user-page">
    <PageTitle --pagetitle-margin-bottom="0">User</PageTitle>

    <div class="account-grid">
        <section class="panel">
            <div class="panel-header">
                <UserRound size={18} />
                <div>
                    <h2>Profile</h2>
                    <p>Update the username used for dashboard sign-in.</p>
                </div>
            </div>

            <Form onSubmit={saveUsername}>
                <div class="form-stack">
                    <TextInput
                        label="Username"
                        bind:value={username}
                        placeholder="admin"
                        maxCharacters={64}
                        disabled={savingUsername || auth.loading}
                    />

                    {#if usernameError}
                        <ErrorBox>{usernameError}</ErrorBox>
                    {/if}

                    <div class="actions">
                        <Button type="submit" disabled={!usernameChanged || savingUsername}>
                            <span class="button-inner">
                                {#if savingUsername}
                                    <RefreshCw size={16} class="spin" />
                                    Saving...
                                {:else}
                                    <Save size={16} />
                                    Save Username
                                {/if}
                            </span>
                        </Button>
                    </div>
                </div>
            </Form>
        </section>

        <section class="panel">
            <div class="panel-header">
                <KeyRound size={18} />
                <div>
                    <h2>Password</h2>
                    <p>Change your password and keep the current session active.</p>
                </div>
            </div>

            <Form onSubmit={savePassword}>
                <div class="form-stack">
                    <TextInput
                        label="Current Password"
                        bind:value={currentPassword}
                        placeholder="Current password"
                        censor={true}
                        maxCharacters={128}
                        boxWidthCh={30}
                        disabled={changingPassword}
                    />
                    <TextInput
                        label="New Password"
                        bind:value={newPassword}
                        placeholder="New password"
                        censor={true}
                        maxCharacters={128}
                        boxWidthCh={30}
                        disabled={changingPassword}
                    />
                    <TextInput
                        label="Confirm Password"
                        bind:value={confirmPassword}
                        placeholder="Repeat new password"
                        censor={true}
                        maxCharacters={128}
                        boxWidthCh={30}
                        disabled={changingPassword}
                    />

                    {#if passwordError}
                        <ErrorBox>{passwordError}</ErrorBox>
                    {/if}

                    <div class="actions">
                        <Button type="submit" disabled={!passwordFormFilled || changingPassword}>
                            <span class="button-inner">
                                {#if changingPassword}
                                    <RefreshCw size={16} class="spin" />
                                    Updating...
                                {:else}
                                    <KeyRound size={16} />
                                    Update Password
                                {/if}
                            </span>
                        </Button>
                    </div>
                </div>
            </Form>
        </section>
    </div>
</main>

<style>
    .user-page {
        display: flex;
        flex-direction: column;
        gap: 1.25rem;
        min-height: 100%;
    }

    .account-grid {
        display: grid;
        grid-template-columns: repeat(auto-fit, minmax(min(100%, 360px), 1fr));
        gap: 1rem;
        width: min(100%, 980px);
    }

    .panel {
        display: flex;
        flex-direction: column;
        gap: 1.2rem;
        padding: 1.2rem;
        border: 1px solid rgba(255, 255, 255, 0.06);
        border-radius: 8px;
        background-color: var(--primary-500);
    }

    .panel-header {
        display: flex;
        gap: 0.75rem;
        align-items: flex-start;
        color: var(--secondary-300);
    }

    .panel-header h2 {
        color: var(--secondary-300);
        font-size: 0.95rem;
        font-weight: 800;
        line-height: 1.1;
        text-transform: uppercase;
    }

    .panel-header p {
        margin-top: 0.35rem;
        color: rgba(255, 255, 255, 0.52);
        font-size: 0.85rem;
        line-height: 1.4;
    }

    .form-stack {
        display: flex;
        flex-direction: column;
        gap: 1rem;
    }

    .actions {
        display: flex;
        justify-content: flex-end;
    }

    .button-inner {
        display: flex;
        align-items: center;
        justify-content: center;
        gap: 0.45rem;
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
        .account-grid {
            gap: 0.75rem;
        }

        .panel {
            padding: 1rem;
        }

        .actions :global(.btn) {
            width: 100%;
        }
    }
</style>
