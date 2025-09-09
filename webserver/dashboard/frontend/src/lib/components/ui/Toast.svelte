<script lang="ts">
    import { fly } from "svelte/transition";
    import Button from "./input/Button.svelte";
    import VerticalSpacer from "./VerticalSpacer.svelte";

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

    export type ToastProps = BaseProps & ActionProps;
    export type ToastType = ToastProps["type"];

    let { message, type = "action", ...rest }: ToastProps = $props();

    let disabled = $state(false);

    export function disable() {
        disabled = true;
    }
</script>

<div class="toast" transition:fly={{ y: 500, duration: 200 }}>
    <h1 class="toast-message">{message}</h1>
    {#if type === "action"}
        <VerticalSpacer
            --spacer-color="var(--primary-450)"
            --spacer-margin="15px"
            --spacer-width="50%"
        />
        <div class="action-buttons">
            <Button onClick={rest.onPositive} {disabled} --btn-font-weight="600"
                >{rest.positiveText || "Yes"}</Button
            >
            <Button
                onClick={rest.onNegative}
                {disabled}
                --btn-font-weight="600"
                --btn-background-color="var(--primary-450)"
                >{rest.negativeText || "No"}
            </Button>
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

    .toast {
        position: fixed;
        bottom: 50px;
        right: 50%;
        transform: translateX(50%);

        padding: 1.5rem;
        border: 1px solid var(--primary-400);
        border-radius: 25px;

        background-color: var(--primary-300);

        box-shadow: 0px 2px 15px rgba(0, 0, 0, 0.1);
    }
</style>
