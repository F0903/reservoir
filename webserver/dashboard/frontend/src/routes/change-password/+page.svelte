<script lang="ts">
    import { goto } from "$app/navigation";
    import { changePassword } from "$lib/api/auth/auth";
    import ErrorBox from "$lib/components/ui/ErrorBox.svelte";
    import Button from "$lib/components/ui/input/Button.svelte";
    import TextInput from "$lib/components/ui/input/TextInput.svelte";
    import VerticalSpacer from "$lib/components/ui/VerticalSpacer.svelte";
    import { log } from "$lib/utils/logger";
    import { getContext } from "svelte";
    import type { PageProps } from "./$types";
    import type { ToastProvider } from "$lib/providers/toast-provider.svelte";
    import { UnauthorizedError } from "$lib/api/api-methods";
    import Form from "$lib/components/ui/input/Form.svelte";
    import { resolve } from "$app/paths";

    let { data }: PageProps = $props();

    const toast: ToastProvider = getContext("toast");

    var error = $state<string | null>(null);
    var processing = $state(false);
    var oldPassword = $state("");
    var newPassword = $state("");

    async function onSubmit() {
        if (oldPassword === newPassword) {
            error = "New password must be different from old password!";
            return;
        }

        try {
            error = null;
            processing = true;

            const returnTo = data.return ?? "/dashboard";
            log.debug("Return to: ", returnTo);

            await changePassword(oldPassword, newPassword);
            toast.show({
                type: "info",
                message: "Password changed successfully.",
                durationMs: 5000,
                dismissText: "Dismiss",
            });

            log.debug("Password reset successful, redirecting...");
            let returnToBase = resolve("/");
            returnToBase += returnTo.startsWith("/") ? returnTo.substring(1) : returnTo;
            await goto(returnToBase, { replaceState: true, invalidateAll: true });
            log.debug("Redirected to ", returnTo);
        } catch (err) {
            if (err instanceof UnauthorizedError) {
                throw err; // If we are unaruthorized here, something went wrong, and we just want to redirect to login like elsewhere.
            }
            error = err instanceof Error ? err.message : String(err);
        } finally {
            processing = false;
        }
    }
</script>

<main class="page-container">
    <div class="change-password-container">
        <h1 class="title">Change Password</h1>
        {#if data.required}
            <p class="required-message">
                For safety reasons, your password must be changed before you can continue.
            </p>
        {/if}
        <Form {onSubmit}>
            <TextInput
                label="Old Password"
                bind:value={oldPassword}
                placeholder="Enter your old password"
                censor={true}
                disabled={processing}
            />
            <TextInput
                label="New Password"
                bind:value={newPassword}
                placeholder="Enter your new password"
                censor={true}
                disabled={processing}
            />
            <VerticalSpacer --spacer-color="var(--primary-500)" --spacer-margin="0px" />
            <div class="bottom-section">
                {#if error}
                    <ErrorBox>{error}</ErrorBox>
                {/if}
                <Button onClick={onSubmit} --btn-margin="auto" disabled={processing}>Reset</Button>
            </div>
        </Form>
    </div>
</main>

<style>
    .required-message {
        color: var(--secondary-300);
        font-weight: 600;
        font-size: 0.95rem;

        margin-top: 1rem;
        margin-bottom: 1rem;

        background-color: var(--primary-500);
        padding: 0.5rem;
        margin-left: auto;
        margin-right: auto;
        border-radius: 8px;
        border: 1px solid var(--primary-400);
    }

    .bottom-section {
        margin-top: 1.2rem;
        display: flex;
        flex-direction: column;
        gap: 1rem;
    }

    .change-password-container {
        max-width: 500px;
        padding: 2rem;
        border-radius: 20px;

        background-color: var(--primary-450);
        color: var(--tertiary-400);
        box-shadow: 0 0 15px 10px rgba(0, 0, 0, 0.1);
    }

    h1 {
        font-size: 2rem;
    }

    .page-container {
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;

        height: 100%;
        width: 100%;
    }
</style>
