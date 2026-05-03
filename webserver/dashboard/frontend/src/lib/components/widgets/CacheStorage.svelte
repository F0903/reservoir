<script lang="ts">
    import { clearCache } from "$lib/api/objects/cache/cache";
    import { userIsAdmin } from "$lib/auth/permissions";
    import Loadable from "$lib/components/ui/Loadable.svelte";
    import Tooltip from "$lib/components/ui/Tooltip.svelte";
    import { getAuthProvider, getMetricsProvider, getToastProvider } from "$lib/context";
    import { formatBytesToLargest } from "$lib/utils/bytestring";
    import { log } from "$lib/utils/logger";
    import { Database, HardDrive, MemoryStick, RefreshCw, Trash2 } from "@lucide/svelte";
    import Widget from "./base/Widget.svelte";
    import CapacityMetricCard from "./utils/CapacityMetricCard.svelte";
    import MetricCard from "./utils/MetricCard.svelte";

    const auth = getAuthProvider();
    const metrics = getMetricsProvider();
    const toast = getToastProvider();

    let clearing = $state(false);
    const status = $derived(metrics.data?.cache.storage ?? null);
    const missingStorageError = $derived(
        metrics.data && !metrics.data.cache.storage
            ? "Cache storage metrics are unavailable."
            : null,
    );
    const error = $derived(metrics.error ?? missingStorageError);
    const loading = $derived(metrics.loading);
    const isAdmin = $derived(userIsAdmin(auth.user));
    const clearDisabled = $derived(!isAdmin || loading || clearing || (status?.entries ?? 0) === 0);
    const clearTooltip = $derived(
        !isAdmin
            ? "Administrator access required"
            : (status?.entries ?? 0) === 0
              ? "Cache is empty"
              : "Clear cache",
    );

    const activeLimitBytes = $derived(
        status
            ? status.type === "memory"
                ? Math.min(status.max_bytes, status.memory_cap_bytes ?? status.max_bytes)
                : status.max_bytes
            : 0,
    );
    const fillPercent = $derived(
        status && activeLimitBytes > 0 ? Math.min(100, (status.bytes / activeLimitBytes) * 100) : 0,
    );

    async function clearCurrentCache() {
        if (!isAdmin) {
            toast.error("Administrator access required.");
            return;
        }
        if (!window.confirm("Clear cached package data?")) {
            return;
        }

        clearing = true;
        try {
            await clearCache();
            toast.success("Cache cleared.");
            await metrics.refreshMetrics();
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
            <Tooltip text={clearTooltip} align="end">
                <button
                    class="tool-button danger"
                    onclick={clearCurrentCache}
                    disabled={clearDisabled}
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
            {@const isMemoryBackend = data.type === "memory"}
            {@const cardActiveLimitBytes = isMemoryBackend
                ? Math.min(data.max_bytes, data.memory_cap_bytes ?? data.max_bytes)
                : data.max_bytes}
            {@const footerItems = [
                ...(isMemoryBackend &&
                data.memory_cap_bytes !== undefined &&
                cardActiveLimitBytes !== data.max_bytes
                    ? [{ label: "Active Limit", value: formatBytesToLargest(cardActiveLimitBytes) }]
                    : []),
                {
                    label: isMemoryBackend ? "Cache Max" : "Storage Limit",
                    value: formatBytesToLargest(data.max_bytes),
                },
                ...(isMemoryBackend && data.memory_cap_bytes !== undefined
                    ? [
                          {
                              label: "Memory Budget",
                              value: formatBytesToLargest(data.memory_cap_bytes),
                          },
                      ]
                    : []),
            ]}
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

                <CapacityMetricCard
                    label="Used"
                    value={formatBytesToLargest(data.bytes)}
                    percent={fillPercent}
                    progressLabel={isMemoryBackend ? "Memory cache fill" : "File cache fill"}
                    {footerItems}
                />
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
    }
</style>
