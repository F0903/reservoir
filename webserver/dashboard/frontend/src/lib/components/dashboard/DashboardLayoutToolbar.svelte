<script lang="ts">
    import Tooltip from "$lib/components/ui/Tooltip.svelte";
    import { Check, Pencil, RefreshCw, RotateCcw } from "@lucide/svelte";

    let {
        editing,
        refreshing = false,
        onEdit,
        onRefresh,
        onReset,
        onSave,
    }: {
        editing: boolean;
        refreshing?: boolean;
        onEdit: () => void;
        onRefresh?: () => void | Promise<void>;
        onReset: () => void;
        onSave: () => void;
    } = $props();
</script>

<div class="dashboard-toolbar" aria-label="Dashboard layout controls">
    {#if onRefresh}
        <Tooltip text="Refresh dashboard metrics" align="end">
            <button
                class="dashboard-action"
                onclick={onRefresh}
                disabled={refreshing}
                aria-label="Refresh dashboard metrics"
            >
                <RefreshCw size={15} class={refreshing ? "spin" : ""} />
                <span>Refresh</span>
            </button>
        </Tooltip>
    {/if}

    {#if editing}
        <Tooltip text="Reset dashboard layout" align="end">
            <button class="dashboard-action" onclick={onReset} aria-label="Reset dashboard layout">
                <RotateCcw size={15} />
                <span>Reset</span>
            </button>
        </Tooltip>
        <Tooltip text="Save dashboard layout" align="end">
            <button
                class="dashboard-action primary"
                onclick={onSave}
                aria-label="Save dashboard layout"
            >
                <Check size={15} />
                <span>Save</span>
            </button>
        </Tooltip>
    {:else}
        <Tooltip text="Edit dashboard layout" align="end">
            <button class="dashboard-action" onclick={onEdit} aria-label="Edit dashboard layout">
                <Pencil size={15} />
                <span>Edit layout</span>
            </button>
        </Tooltip>
    {/if}
</div>

<style>
    .dashboard-toolbar {
        display: flex;
        justify-content: flex-end;
        align-items: center;
        gap: 0.5rem;
        min-height: 2rem;
        margin-bottom: 0.75rem;
    }

    .dashboard-action {
        display: inline-flex;
        align-items: center;
        gap: 0.4rem;
        min-height: 2rem;
        padding: 0.35rem 0.65rem;
        border: 1px solid rgba(255, 255, 255, 0.06);
        border-radius: 8px;
        background-color: rgba(255, 255, 255, 0.025);
        color: rgba(255, 255, 255, 0.65);
        font-size: 0.72rem;
        font-weight: 700;
        transition:
            background-color 120ms ease,
            color 120ms ease,
            border-color 120ms ease,
            transform 120ms ease;
    }

    .dashboard-action:hover:enabled {
        transform: translateY(-1px);
        border-color: rgba(255, 255, 255, 0.12);
        background-color: rgba(255, 255, 255, 0.05);
        color: var(--secondary-300);
    }

    .dashboard-action:disabled {
        cursor: default;
        opacity: 0.42;
    }

    .dashboard-action.primary {
        color: var(--secondary-300);
        border-color: color-mix(in srgb, var(--secondary-300) 30%, transparent);
        background-color: color-mix(in srgb, var(--secondary-800) 28%, transparent);
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
        .dashboard-toolbar {
            margin: 0 0 0.75rem;
        }

        .dashboard-action span {
            display: none;
        }
    }
</style>
