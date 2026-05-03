<script lang="ts">
    import type { Snippet } from "svelte";

    type ButtonType = "button" | "submit" | "reset";

    let {
        onClick,
        disabled = false,
        type = "button",
        icon,
        children,
    }: {
        onClick?: (_event: MouseEvent) => void | Promise<void>;
        disabled?: boolean;
        type?: ButtonType;
        icon?: Snippet;
        children: Snippet;
    } = $props();
</script>

<button class="panel-action-button" onclick={onClick} {disabled} {type}>
    <span class="button-inner">
        {#if icon}
            <span class="icon" aria-hidden="true">
                {@render icon()}
            </span>
        {/if}
        <span class="label">
            {@render children()}
        </span>
    </span>
</button>

<style>
    .panel-action-button {
        min-height: 2.2rem;
        padding: 0.35rem 0.42rem 0.35rem 0.6rem;
        border: 1px solid color-mix(in srgb, var(--secondary-300) 22%, transparent);
        border-radius: 8px;
        background-color: color-mix(in srgb, var(--secondary-800) 24%, var(--primary-600));
        box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.04);
        color: var(--secondary-300);
        cursor: pointer;
        font-size: 0.78rem;
        font-weight: 800;
        transition:
            border-color 120ms ease,
            background-color 120ms ease,
            color 120ms ease,
            transform 120ms ease;
    }

    .panel-action-button:hover:enabled {
        border-color: color-mix(in srgb, var(--secondary-300) 38%, transparent);
        background-color: color-mix(in srgb, var(--secondary-800) 34%, var(--primary-600));
        color: var(--secondary-200);
    }

    .panel-action-button:active:enabled {
        transform: translateY(1px);
    }

    .panel-action-button:disabled {
        border-color: rgba(255, 255, 255, 0.07);
        background-color: var(--primary-500);
        color: rgba(255, 255, 255, 0.34);
        cursor: default;
    }

    .button-inner {
        display: flex;
        align-items: center;
        justify-content: center;
        gap: 0.5rem;
    }

    .icon {
        display: grid;
        place-items: center;
        width: 1.35rem;
        height: 1.35rem;
        border-radius: 6px;
        background-color: color-mix(in srgb, var(--secondary-300) 14%, transparent);
        color: var(--secondary-300);
    }

    .panel-action-button:disabled .icon {
        background-color: rgba(255, 255, 255, 0.04);
        color: rgba(255, 255, 255, 0.28);
    }

    .label {
        line-height: 1.2;
    }
</style>
