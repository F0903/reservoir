<script lang="ts">
    import { fly } from "svelte/transition";
    import Button from "./input/Button.svelte";
    import { onMount } from "svelte";
    import { ToastHandle } from "$lib/providers/toast/toast-provider.svelte";
    import { log } from "$lib/utils/logger";
    import { Info, CircleCheckBig, CircleAlert, CircleQuestionMark } from "@lucide/svelte";

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

    type BaseTextProps = {
        durationMs: number;
        dismissText?: string;
        onDismiss?: () => Promise<void>;
    };

    type InfoProps = {
        type: "info";
    } & BaseTextProps;

    type SuccessProps = {
        type: "success";
    } & BaseTextProps;

    type ErrorProps = {
        type: "error";
    } & BaseTextProps;

    export type ToastProps =
        | (BaseProps & ActionProps)
        | (BaseProps & InfoProps)
        | (BaseProps & SuccessProps)
        | (BaseProps & ErrorProps);
    export type ToastType = ToastProps["type"];

    let props: ToastProps & { handle: ToastHandle } = $props();

    let disabled = $state(false);

    onMount(() => {
        if (props.type === "info" || props.type === "error" || props.type === "success") {
            const timeout = setTimeout(async () => {
                await disableAndDo(props.onDismiss)();
            }, props.durationMs);
            return () => clearTimeout(timeout);
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
    transition:fly={{ y: 20, duration: 250 }}
    role={props.type === "error" ? "alert" : "status"}
    aria-live={props.type === "error" ? "assertive" : "polite"}
>
    <div class="accent-bar"></div>
    <div class="toast-content">
        <div class="icon-container">
            {#if props.type === "success"}
                <CircleCheckBig size={24} />
            {:else if props.type === "error"}
                <CircleAlert size={24} />
            {:else if props.type === "action"}
                <CircleQuestionMark size={24} />
            {:else}
                <Info size={24} />
            {/if}
        </div>

        <div class="text-and-actions">
            <h1 class="toast-message">{props.message}</h1>

            {#if props.type === "action"}
                <div class="action-buttons">
                    <Button
                        onClick={disableAndDo(props.onPositive)}
                        {disabled}
                        --btn-font-weight="600"
                        --btn-border-radius="8px"
                        --btn-background-color="var(--tertiary-500)"
                    >
                        {props.positiveText || "Yes"}
                    </Button>
                    <Button
                        onClick={disableAndDo(props.onNegative)}
                        {disabled}
                        --btn-font-weight="600"
                        --btn-border-radius="8px"
                        --btn-background-color="var(--primary-300)"
                        --btn-text-color="var(--text-400)"
                    >
                        {props.negativeText || "No"}
                    </Button>
                </div>
            {:else}
                <div class="action-buttons single">
                    <Button
                        onClick={disableAndDo(props.onDismiss)}
                        {disabled}
                        --btn-font-weight="600"
                        --btn-border-radius="8px"
                        --btn-padding="0.3rem 0.6rem"
                        --btn-font-size="0.85rem"
                        --btn-background-color="rgba(255, 255, 255, 0.1)"
                        --btn-text-color="var(--text-400)"
                    >
                        {props.dismissText || (props.type === "error" ? "Dismiss" : "OK")}
                    </Button>
                </div>
            {/if}
        </div>
    </div>
</div>

<style>
    .toast {
        --toast-bg-color: var(--primary-400);
        --toast-accent-color: var(--secondary-400);

        position: relative;
        overflow: hidden;

        background-color: var(--toast-bg-color);
        backdrop-filter: blur(10px);
        -webkit-backdrop-filter: blur(10px);

        border: 1px solid rgba(255, 255, 255, 0.1);
        border-radius: 12px;

        box-shadow: 0 8px 32px rgba(0, 0, 0, 0.4);

        width: 380px;
        pointer-events: auto;
    }

    .toast.success {
        --toast-accent-color: var(--success-color);
    }

    .toast.error {
        --toast-accent-color: var(--error-color);
    }

    .toast.action {
        --toast-accent-color: var(--tertiary-500);
    }

    .accent-bar {
        position: absolute;
        left: 0;
        top: 0;
        bottom: 0;
        width: 6px;
        background-color: var(--toast-accent-color);
    }

    .toast-content {
        display: flex;
        flex-direction: row;
        padding: 1rem 1rem 1rem 1.2rem;
        gap: 1rem;
        align-items: flex-start;
    }

    .icon-container {
        color: var(--toast-accent-color);
        flex-shrink: 0;
        margin-top: 2px;
    }

    .text-and-actions {
        display: flex;
        flex-direction: column;
        gap: 0.75rem;
        flex-grow: 1;
    }

    .toast-message {
        margin: 0;
        font-size: 1rem;
        font-weight: 500;
        color: var(--text-400);
        line-height: 1.4;
        text-align: left;
    }

    .action-buttons {
        display: flex;
        flex-direction: row;
        gap: 0.75rem;
        justify-content: flex-end;
    }

    .action-buttons.single {
        margin-top: 0.25rem;
    }
</style>
