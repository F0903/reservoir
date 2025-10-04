<script lang="ts">
    import { goto } from "$app/navigation";
    import { UnauthorizedError } from "$lib/api/api-methods";
    import { login } from "$lib/api/auth/auth";
    import ErrorBox from "$lib/components/ui/ErrorBox.svelte";
    import Button from "$lib/components/ui/input/Button.svelte";
    import Form from "$lib/components/ui/input/Form.svelte";
    import TextInput from "$lib/components/ui/input/TextInput.svelte";
    import VerticalSpacer from "$lib/components/ui/VerticalSpacer.svelte";
    import { log } from "$lib/utils/logger";
    import type { PageProps } from "./$types";

    let { data }: PageProps = $props();

    let usernameInput: TextInput | undefined = $state();
    let passwordInput: TextInput | undefined = $state();

    let username = $state("");
    let password = $state("");

    let loggingIn = $state(false);
    let error = $state<string | null>(null);

    async function onLogin() {
        try {
            error = null;
            loggingIn = true;

            const returnTo = data.return ?? "/dashboard";

            const user = await login({ username, password });
            if (user.password_reset_required) {
                log.debug("Login successful, password reset required, redirecting...");
                const params = new URLSearchParams();
                params.set("return", returnTo);
                await goto("/reset-password?" + params.toString(), { replaceState: true });
                return;
            }

            log.debug("Login successful, redirecting...");

            await goto(returnTo, { replaceState: true, invalidateAll: true });
            log.debug("Redirected to ", returnTo);
        } catch (err) {
            log.error("Login failed: ", err);
            if (err instanceof UnauthorizedError) {
                error = "Invalid username or password.";
                username = "";
                password = "";
            } else {
                error = err instanceof Error ? err.message : String(err);
            }
        } finally {
            loggingIn = false;
        }
    }
</script>

<div class="page-container">
    <div class="login-container">
        <Form onSubmit={onLogin}>
            <h1>Login</h1>
            <TextInput
                bind:this={usernameInput}
                label="Username"
                bind:value={username}
                placeholder="Enter your username"
                disabled={loggingIn}
            />
            <TextInput
                bind:this={passwordInput}
                label="Password"
                bind:value={password}
                placeholder="Enter your password"
                censor={true}
                disabled={loggingIn}
            />
            <VerticalSpacer --spacer-color="var(--primary-500)" --spacer-margin="0px" />
            <div class="bottom-section">
                {#if error}
                    <ErrorBox>{error}</ErrorBox>
                {/if}
                <Button onClick={onLogin} --btn-margin="auto" disabled={loggingIn}>Login</Button>
            </div>
        </Form>
    </div>
</div>

<style>
    .bottom-section {
        margin-top: 1.2rem;
        display: flex;
        flex-direction: column;
        gap: 1rem;
    }

    .login-container {
        background-color: var(--primary-450);
        color: var(--tertiary-400);
        padding: 2rem;
        border-radius: 20px;
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
