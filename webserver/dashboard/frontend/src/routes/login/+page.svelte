<script lang="ts">
    import { goto } from "$app/navigation";
    import { resolve } from "$app/paths";
    import ErrorBox from "$lib/components/ui/ErrorBox.svelte";
    import Button from "$lib/components/ui/input/Button.svelte";
    import Form from "$lib/components/ui/input/Form.svelte";
    import TextInput from "$lib/components/ui/input/TextInput.svelte";
    import VerticalSpacer from "$lib/components/ui/VerticalSpacer.svelte";
    import { log } from "$lib/utils/logger";
    import type { PageProps } from "./$types";
    import { getAuthProvider } from "$lib/context";
    import UnauthorizedError from "$lib/api/unauthorized-error";
    import { User, Lock, ArrowRight } from "@lucide/svelte";

    let { data }: PageProps = $props();

    const auth = getAuthProvider();

    let username = $state("");
    let password = $state("");

    let loggingIn = $state(false);
    let error = $state<string | null>(null);

    async function onLogin() {
        try {
            error = null;
            loggingIn = true;

            const returnTo = data.return ?? "/dashboard";
            log.debug("Return to: ", returnTo);

            const user = await auth.login({ username, password });
            log.debug("Got user from login: ", user);

            if (user.password_change_required) {
                log.debug("Login successful, password change required, redirecting...");

                // We don't need SvelteURLSearchParams here, since we're not using it reactively in the UI.
                // eslint-disable-next-line svelte/prefer-svelte-reactivity
                const params = new URLSearchParams();

                const pathToReturn = returnTo.startsWith("/") ? returnTo.substring(1) : returnTo;
                params.set("return", pathToReturn);
                params.set("required", "true");
                let url = resolve("/change-password");
                url += `?${params.toString()}`;

                await goto(url, { replaceState: true });
                return;
            }

            log.debug("Login successful, redirecting...");
            let returnToBase = resolve("/");
            returnToBase += returnTo.startsWith("/") ? returnTo.substring(1) : returnTo;
            await goto(returnToBase, { replaceState: true, invalidateAll: true });
            log.debug("Redirected to ", returnTo);
        } catch (err) {
            log.error("Login failed: ", err);
            if (err instanceof UnauthorizedError) {
                error = "Invalid username or password.";
                password = "";
            } else {
                error = err instanceof Error ? err.message : String(err);
            }
        } finally {
            loggingIn = false;
        }
    }
</script>

<main class="login-page">
    <div class="background-glow"></div>

    <div class="login-container">
        <div class="brand-section">
            <h1 class="logo-text">reservoir</h1>
        </div>

        <Form onSubmit={onLogin}>
            <div class="input-group">
                <TextInput
                    label="Username"
                    bind:value={username}
                    placeholder="admin"
                    disabled={loggingIn}
                >
                    {#snippet suffixElement()}
                        <div class="field-icon"><User size={18} /></div>
                    {/snippet}
                </TextInput>
            </div>

            <div class="input-group">
                <TextInput
                    label="Password"
                    bind:value={password}
                    placeholder="••••••••"
                    censor={true}
                    disabled={loggingIn}
                >
                    {#snippet suffixElement()}
                        <div class="field-icon"><Lock size={18} /></div>
                    {/snippet}
                </TextInput>
            </div>

            <VerticalSpacer --spacer-color="rgba(255,255,255,0.05)" --spacer-margin="1rem 0" />

            <div class="bottom-section">
                {#if error}
                    <div class="error-wrapper">
                        <ErrorBox>{error}</ErrorBox>
                    </div>
                {/if}

                <Button
                    onClick={onLogin}
                    disabled={loggingIn}
                    --btn-width="100%"
                    --btn-padding="0.8rem"
                    --btn-border-radius="12px"
                >
                    <div class="btn-content">
                        <span>{loggingIn ? "Authenticating..." : "Login"}</span>
                        {#if !loggingIn}
                            <ArrowRight size={18} />
                        {/if}
                    </div>
                </Button>
            </div>
        </Form>
    </div>
</main>

<style>
    .login-page {
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        height: 100%;
        width: 100%;
        background-color: var(--primary-800);
        position: relative;
        overflow: hidden;
    }

    .background-glow {
        position: absolute;
        width: 600px;
        height: 600px;
        background: radial-gradient(circle, var(--secondary-800) 0%, transparent 70%);
        opacity: 0.3;
        top: 50%;
        left: 50%;
        transform: translate(-50%, -50%);
        pointer-events: none;
    }

    .login-container {
        width: 100%;
        max-width: 420px;
        background-color: var(--primary-450);
        backdrop-filter: blur(20px);
        -webkit-backdrop-filter: blur(20px);
        padding: 2.5rem;
        border-radius: 24px;
        border: 1px solid rgba(255, 255, 255, 0.05);
        box-shadow: 0 20px 50px rgba(0, 0, 0, 0.5);
        z-index: 1;
    }

    .brand-section {
        text-align: center;
        margin-bottom: 2rem;
    }

    .logo-text {
        font-size: 2.5rem;
        font-weight: 800;
        color: var(--tertiary-400);
        letter-spacing: -0.02em;
        margin-bottom: 0.25rem;
    }

    .input-group {
        position: relative;
        margin-bottom: 0.5rem;
    }

    .field-icon {
        color: var(--secondary-400);
        opacity: 0.5;
        pointer-events: none;
        margin-top: 3px;
    }

    .bottom-section {
        margin-top: 0.5rem;
        display: flex;
        flex-direction: column;
        gap: 1.5rem;
    }

    .btn-content {
        display: flex;
        align-items: center;
        justify-content: center;
        gap: 0.5rem;
    }

    .error-wrapper {
        animation: shake 0.4s cubic-bezier(0.36, 0.07, 0.19, 0.97) both;
    }

    @keyframes shake {
        10%,
        90% {
            transform: translate3d(-1px, 0, 0);
        }
        20%,
        80% {
            transform: translate3d(2px, 0, 0);
        }
        30%,
        50%,
        70% {
            transform: translate3d(-4px, 0, 0);
        }
        40%,
        60% {
            transform: translate3d(4px, 0, 0);
        }
    }
</style>
