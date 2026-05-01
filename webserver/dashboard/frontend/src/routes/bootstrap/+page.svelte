<script lang="ts">
    import { goto } from "$app/navigation";
    import { resolve } from "$app/paths";
    import UnauthorizedError from "$lib/api/unauthorized-error";
    import ErrorBox from "$lib/components/ui/ErrorBox.svelte";
    import Button from "$lib/components/ui/input/Button.svelte";
    import Form from "$lib/components/ui/input/Form.svelte";
    import TextInput from "$lib/components/ui/input/TextInput.svelte";
    import VerticalSpacer from "$lib/components/ui/VerticalSpacer.svelte";
    import { getAuthProvider, getToastProvider } from "$lib/context";
    import { log } from "$lib/utils/logger";
    import { ArrowRight, Lock, UserPlus } from "@lucide/svelte";
    import { onMount } from "svelte";

    const auth = getAuthProvider();
    const toast = getToastProvider();

    let username = $state("admin");
    let password = $state("");
    let confirmPassword = $state("");
    let loading = $state(true);
    let processing = $state(false);
    let error = $state<string | null>(null);

    onMount(async () => {
        try {
            const status = await auth.bootstrapStatus();
            if (!status.bootstrap_required) {
                await goto(resolve("/dashboard"), { replaceState: true });
                return;
            }
        } catch (err) {
            log.error("Failed to check bootstrap status:", err);
            error = err instanceof Error ? err.message : String(err);
        } finally {
            loading = false;
        }
    });

    async function onBootstrap() {
        if (loading || processing) {
            return;
        }
        if (password.length < 12) {
            error = "Password must be at least 12 characters.";
            return;
        }
        if (password !== confirmPassword) {
            error = "Passwords do not match.";
            confirmPassword = "";
            return;
        }

        try {
            error = null;
            processing = true;

            await auth.bootstrap({ username, password });
            toast.success("Admin account created.");
            await goto(resolve("/dashboard"), { replaceState: true, invalidateAll: true });
        } catch (err) {
            log.error("Bootstrap setup failed:", err);
            if (err instanceof UnauthorizedError) {
                throw err;
            }
            error = err instanceof Error ? err.message : String(err);
        } finally {
            processing = false;
        }
    }
</script>

<main class="bootstrap-page">
    <div class="background-glow"></div>

    <div class="bootstrap-container">
        <div class="brand-section">
            <h1 class="logo-text">reservoir</h1>
            <p class="setup-text">Create the initial admin account.</p>
        </div>

        <Form onSubmit={onBootstrap}>
            <TextInput label="Username" bind:value={username} disabled={loading || processing}>
                {#snippet suffixElement()}
                    <div class="field-icon"><UserPlus size={18} /></div>
                {/snippet}
            </TextInput>

            <TextInput
                label="Password"
                bind:value={password}
                censor={true}
                min={12}
                disabled={loading || processing}
            >
                {#snippet suffixElement()}
                    <div class="field-icon"><Lock size={18} /></div>
                {/snippet}
            </TextInput>

            <TextInput
                label="Confirm Password"
                bind:value={confirmPassword}
                censor={true}
                min={12}
                disabled={loading || processing}
            >
                {#snippet suffixElement()}
                    <div class="field-icon"><Lock size={18} /></div>
                {/snippet}
            </TextInput>

            <VerticalSpacer --spacer-color="rgba(255,255,255,0.05)" --spacer-margin="1rem 0" />

            <div class="bottom-section">
                {#if error}
                    <ErrorBox>{error}</ErrorBox>
                {/if}

                <Button
                    type="submit"
                    disabled={loading || processing}
                    --btn-width="100%"
                    --btn-padding="0.8rem"
                    --btn-border-radius="12px"
                >
                    <div class="btn-content">
                        <span>{processing ? "Creating..." : "Create admin"}</span>
                        {#if !processing}
                            <ArrowRight size={18} />
                        {/if}
                    </div>
                </Button>
            </div>
        </Form>
    </div>
</main>

<style>
    .bootstrap-page {
        display: flex;
        align-items: center;
        justify-content: center;
        height: 100%;
        width: 100%;
        background-color: var(--primary-800);
        position: relative;
        overflow: hidden;
        padding: 1rem;
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

    .bootstrap-container {
        width: 100%;
        max-width: 440px;
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
        margin-bottom: 0.25rem;
    }

    .setup-text {
        color: var(--secondary-300);
        font-size: 0.95rem;
        font-weight: 600;
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
</style>
