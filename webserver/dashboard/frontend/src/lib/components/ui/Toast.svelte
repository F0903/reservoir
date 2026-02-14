<script lang="ts">
    import { fly } from "svelte/transition";
    // Mock fly for testing if needed, but let's try to fix it in test-setup first
    import Button from "./input/Button.svelte";
    import VerticalSpacer from "./VerticalSpacer.svelte";
    import { onMount } from "svelte";
    import type { ToastHandle } from "$lib/providers/toast-provider.svelte";
    import { log } from "$lib/utils/logger";

    type BaseProps = {
        message: string;
    };

    type ActionProps = {
        type: "action";
        positiveText?: string;
        negativeText?: string;
        onPositive?: () => Promise<void>;
        onNegative?: () => Promise<void>;
    };

    type InfoProps = {
        type: "info";
        durationMs: number;
        dismissText?: string;
        onDismiss?: () => Promise<void>;
    };

    type ErrorProps = {
        type: "error";
        durationMs: number;
        dismissText?: string;
        onDismiss?: () => Promise<void>;
    };

    export type ToastProps =
        | (BaseProps & ActionProps)
        | (BaseProps & InfoProps)
        | (BaseProps & ErrorProps);
    export type ToastType = ToastProps["type"];

    let props: ToastProps & { handle: ToastHandle } = $props();

    let disabled = $state(false);

    onMount(() => {
        if (props.type === "info" || props.type === "error") {
            setTimeout(async () => {
                await disableAndDo(props.onDismiss)();
            }, props.durationMs);
        }
    });

    function disableAndDo(fn?: () => Promise<void>): () => Promise<void> {
        return async () => {
            disabled = true;
            try {
                await fn?.();
            } catch (e) {
                log.error("Toast action failed:", e);
                disabled = false; // Re-enable so user can try again or dismiss
                return;
            }
            props.handle.close();
        };
    }

    export function disable() {
        disabled = true;
    }
</script>

<div
    class="toast {props.type}"
    transition:fly={{ y: 500, duration: 200 }}
    role={props.type === "error" ? "alert" : "status"}
    aria-live={props.type === "error" ? "assertive" : "polite"}
>
    <h1 class="toast-message">{props.message}</h1>
    {#if props.type === "action"}
        <VerticalSpacer
            --spacer-color="var(--primary-450)"
            --spacer-margin="15px"
            --spacer-width="50%"
        />
        <div class="action-buttons">
            <Button onClick={disableAndDo(props.onPositive)} {disabled} --btn-font-weight="600"
                >{props.positiveText || "Yes"}</Button
            >
            <Button
                onClick={disableAndDo(props.onNegative)}
                {disabled}
                --btn-font-weight="600"
                --btn-background-color="var(--secondary-600)"
                >{props.negativeText || "No"}
            </Button>
        </div>
    {:else if props.type === "info"}
        <VerticalSpacer
            --spacer-color="var(--primary-450)"
            --spacer-margin="15px"
            --spacer-width="50%"
        />
        <div class="action-buttons">
            <Button onClick={disableAndDo(props.onDismiss)} {disabled} --btn-font-weight="600"
                >{props.dismissText || "OK"}</Button
            >
        </div>
    {:else if props.type === "error"}
        <VerticalSpacer
            --spacer-color="var(--primary-450)"
            --spacer-margin="15px"
            --spacer-width="50%"
        />
        <div class="action-buttons">
            <Button onClick={disableAndDo(props.onDismiss)} {disabled} --btn-font-weight="600"
                >{props.dismissText || "Dismiss"}</Button
            >
        </div>
    {/if}
</div>

<style>
    .action-buttons {
        display: flex;
        flex-direction: row;
        margin: auto;
        justify-content: space-evenly;
        width: 100%;
    }

    .toast-message {
        margin: 0;
        font-size: 1.2rem;
        font-weight: 400;
        margin-bottom: 0px;
    }

    .toast.error {
        border-color: var(--error-border-color);
        background-color: var(--error-background-color);
        color: var(--error-text-color);

        --btn-background-color: var(--secondary-600);
    }

    .toast {
        --toast-bg-color: var(--primary-300);
        --toast-border-color: var(--primary-400);

        position: fixed;
        bottom: 50px;
        right: 50%;
        transform: translateX(50%);

        text-align: center;

        padding: 1.5rem;
        border: 2px solid var(--toast-border-color);
        border-radius: 25px;

        background-color: var(--toast-bg-color);

        box-shadow: 0px 2px 15px rgba(0, 0, 0, 0.1);
    }
</style>
