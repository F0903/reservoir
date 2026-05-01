<script lang="ts">
    import { clearCache, getCacheStatus, type CacheStatus } from "$lib/api/objects/cache/cache";
    import Loadable from "$lib/components/ui/Loadable.svelte";
    import Tooltip from "$lib/components/ui/Tooltip.svelte";
    import { getMetricsProvider, getToastProvider } from "$lib/context";
    import { formatBytesToLargest } from "$lib/utils/bytestring";
    import { log } from "$lib/utils/logger";
    import { Database, HardDrive, MemoryStick, RefreshCw, Trash2 } from "@lucide/svelte";
    import { onMount } from "svelte";
    import Widget from "./base/Widget.svelte";
    import MetricCard from "./utils/MetricCard.svelte";

    const metrics = getMetricsProvider();
    const toast = getToastProvider();

    let status = $state<CacheStatus | null>(null);
    let error = $state<string | null>(null);
    let loading = $state(false);
    let clearing = $state(false);

    const fillPercent = $derived(
        status && status.max_bytes > 0 ? Math.min(100, (status.bytes / status.max_bytes) * 100) : 0,
    );

    onMount(() => {
        refreshStatus();
    });

    async function refreshStatus() {
        loading = true;
        try {
            status = { ...(await getCacheStatus()) };
            error = null;
        } catch (err) {
            log.error("Failed to refresh cache status:", err);
            error = err instanceof Error ? err.message : String(err);
        } finally {
            loading = false;
        }
    }

    async function clearCurrentCache() {
        if (!window.confirm("Clear cached package data?")) {
            return;
        }

        clearing = true;
        try {
            await clearCache();
            toast.success("Cache cleared.");
            await Promise.all([refreshStatus(), metrics.refreshMetrics()]);
        } catch (err) {
            log.error("Failed to clear cache:", err);
            toast.error(err instanceof Error ? err.message : String(err));
        } finally {
            clearing = false;
        }
    }
</script>

<Widget title="Cache Storage">
    {#snippet headerControls()}
        <div class="header-toolbar" aria-label="Cache storage actions">
            <Tooltip text="Refresh cache status" align="end">
                <button
                    class="tool-button"
                    onclick={refreshStatus}
                    disabled={loading || clearing}
                    aria-label="Refresh cache status"
                >
                    <RefreshCw size={14} class={loading ? "spin" : ""} />
                </button>
            </Tooltip>
            <Tooltip
                text={(status?.entries ?? 0) === 0 ? "Cache is empty" : "Clear cache"}
                align="end"
            >
                <button
                    class="tool-button danger"
                    onclick={clearCurrentCache}
                    disabled={loading || clearing || (status?.entries ?? 0) === 0}
                    aria-label="Clear cache"
                >
                    {#if clearing}
                        <RefreshCw size={14} class="spin" />
                    {:else}
                        <Trash2 size={14} />
                    {/if}
                </button>
            </Tooltip>
        </div>
    {/snippet}

    <Loadable state={status} {error}>
        {#snippet children(data)}
            <div class="storage-panel">
                <div class="summary-grid">
                    <MetricCard
                        label="Backend"
                        value={data.type}
                        icon={data.type === "memory" ? MemoryStick : HardDrive}
                    />
                    <MetricCard
                        label="Entries"
                        value={data.entries.toLocaleString()}
                        icon={Database}
                    />
                </div>

                <div class="capacity-panel">
                    <div class="capacity-topline">
                        <div class="used-block">
                            <span class="panel-label">Used</span>
                            <strong>{formatBytesToLargest(data.bytes)}</strong>
                        </div>
                        <span class="fill-percent">{fillPercent.toFixed(1)}%</span>
                    </div>

                    <div class="capacity-track" aria-label="Cache fill">
                        <div class="capacity-fill" style:width={`${fillPercent}%`}></div>
                    </div>

                    <div class="capacity-meta">
                        <span>Max <strong>{formatBytesToLargest(data.max_bytes)}</strong></span>
                        {#if data.memory_cap_bytes !== undefined}
                            <span
                                >Memory <strong
                                    >{formatBytesToLargest(data.memory_cap_bytes)}</strong
                                ></span
                            >
                        {/if}
                    </div>
                </div>
            </div>
        {/snippet}
    </Loadable>
</Widget>

<style>
    .storage-panel {
        display: grid;
        grid-template-rows: auto minmax(0, 1fr);
        gap: 0.75rem;
        height: 100%;
        min-height: 0;
    }

    .summary-grid {
        display: grid;
        grid-template-columns: repeat(2, minmax(0, 1fr));
        gap: 0.6rem;
    }

    .summary-grid :global(.metric-card-wrapper) {
        --metric-padding: 0.55rem 0.65rem;
        --metric-border-radius: 8px;
        --metric-value-size: 0.95rem;
        --metric-label-size: 0.58rem;
        min-height: 3.6rem;
    }

    .capacity-panel {
        display: flex;
        flex-direction: column;
        justify-content: space-around;
        gap: 0.7rem;
        min-height: 0;
        height: 100%;
        padding: 0.8rem;
        border-radius: 8px;
        border: 1px solid var(--primary-500);
        background-color: var(--primary-600);
    }

    .capacity-topline {
        display: flex;
        justify-content: space-between;
        align-items: flex-start;
        gap: 1rem;
    }

    .used-block {
        display: flex;
        flex-direction: column;
        gap: 0.2rem;
        min-width: 0;
    }

    .panel-label {
        color: rgba(255, 255, 255, 0.4);
        font-size: 0.58rem;
        font-weight: 700;
        letter-spacing: 0.08em;
        line-height: 1;
        text-transform: uppercase;
    }

    .used-block strong {
        color: var(--secondary-300);
        font-size: 1.2rem;
        font-weight: 700;
        line-height: 1;
        white-space: nowrap;
    }

    .fill-percent {
        color: var(--secondary-300);
        font-family: "Chivo Mono Variable", monospace;
        font-size: 0.8rem;
        font-weight: 700;
    }

    .capacity-track {
        height: 0.5rem;
        overflow: hidden;
        border-radius: 999px;
        background-color: var(--primary-700);
        border: 1px solid rgba(255, 255, 255, 0.08);
    }

    .capacity-fill {
        height: 100%;
        border-radius: inherit;
        background-color: var(--secondary-300);
        transition: width 160ms ease;
    }

    .capacity-meta {
        display: flex;
        flex-wrap: wrap;
        gap: 0.45rem 0.8rem;
        color: rgba(255, 255, 255, 0.4);
        font-size: 0.68rem;
        font-weight: 600;
    }

    .capacity-meta strong {
        color: var(--secondary-300);
        font-weight: 700;
    }

    .header-toolbar {
        display: flex;
        align-items: center;
        gap: 0.3rem;
    }

    .tool-button {
        display: grid;
        place-items: center;
        width: 1.75rem;
        height: 1.75rem;
        border-radius: 7px;
        border: 1px solid rgba(255, 255, 255, 0.06);
        background-color: rgba(255, 255, 255, 0.025);
        color: rgba(255, 255, 255, 0.58);
        transition:
            background-color 120ms ease,
            color 120ms ease,
            border-color 120ms ease,
            transform 120ms ease;
    }

    .tool-button:hover:enabled {
        transform: translateY(-1px);
        border-color: rgba(255, 255, 255, 0.12);
        background-color: rgba(255, 255, 255, 0.05);
        color: var(--secondary-300);
    }

    .tool-button:active:enabled {
        transform: translateY(0);
    }

    .tool-button:disabled {
        cursor: default;
        opacity: 0.38;
    }

    .tool-button.danger:hover:enabled {
        color: var(--error-color);
        border-color: color-mix(in srgb, var(--error-color) 28%, transparent);
        background-color: color-mix(in srgb, var(--error-bg) 45%, transparent);
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
        .storage-panel {
            gap: 0.6rem;
        }

        .capacity-panel {
            padding: 0.7rem;
        }

        .used-block strong {
            font-size: 1.15rem;
        }
    }
</style>
