<script lang="ts">
    import { goto } from "$app/navigation";
    import ErrorBox from "$lib/components/ui/ErrorBox.svelte";
    import Button from "$lib/components/ui/input/Button.svelte";
    import Form from "$lib/components/ui/input/Form.svelte";
    import TextInput from "$lib/components/ui/input/TextInput.svelte";
    import VerticalSpacer from "$lib/components/ui/VerticalSpacer.svelte";
    import { log } from "$lib/utils/logger";
    import type { PageProps } from "./$types";

    let { data }: PageProps = $props();

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

            //TODO: Call the API to reset the password

            log.debug("Password reset successful, redirecting...");
            await goto(returnTo, { replaceState: true, invalidateAll: true });
            log.debug("Redirected to ", returnTo);
        } catch (err) {
            error = err instanceof Error ? err.message : String(err);
        } finally {
            processing = false;
        }
    }
</script>

<div class="page-container">
    <div class="reset-password-container">
        <h1 class="title">Reset Password</h1>
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
</div>
