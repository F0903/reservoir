<script lang="ts">
    import Loadable from "../ui/Loadable.svelte";
    import { getMetricsProvider } from "$lib/context";
    import Widget from "./base/Widget.svelte";
    import MetricCard from "./utils/MetricCard.svelte";

    const metrics = getMetricsProvider();
</script>

<Widget title="Cache Activity">
    <Loadable state={metrics.data} error={metrics.error}>
        {#snippet children(data)}
            {@const cacheOperations = data.cache.cache_hits + data.cache.cache_misses}
            {@const operationHitRate =
                cacheOperations > 0 ? (data.cache.cache_hits / cacheOperations) * 100 : 0}

            <div class="activity-grid">
                <MetricCard
                    label="Operation Hits"
                    value={data.cache.cache_hits.toLocaleString()}
                    --metric-value-color="var(--success-color)"
                />
                <MetricCard
                    label="Operation Misses"
                    value={data.cache.cache_misses.toLocaleString()}
                />
                <MetricCard
                    label="Operation Errors"
                    value={data.cache.cache_errors.toLocaleString()}
                    --metric-value-color="var(--log-error-color)"
                />
                <MetricCard label="Operation Hit Rate" value={`${operationHitRate.toFixed(1)}%`} />
            </div>
        {/snippet}
    </Loadable>
</Widget>

<style>
    .activity-grid {
        display: grid;
        grid-template-columns: repeat(2, minmax(0, 1fr));
        gap: 0.75rem;
        height: 100%;
        min-height: 0;
    }
</style>
