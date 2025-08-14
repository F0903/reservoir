<script lang="ts">
    import Chart from "chart.js/auto";
    import { onDestroy, onMount } from "svelte";
    import type { ChartConfiguration, ChartData, ChartType } from "chart.js";
    import { customChartColors } from "$lib/utils/chart-colors";
    import { deepMerge } from "$lib/utils/objects";
    import { log } from "$lib/utils/logger";

    // Register the custom color plugin
    Chart.register(customChartColors);

    let {
        type,
        data = $bindable(),
        options = {},
    }: { type: ChartType; data: ChartData; options?: ChartConfiguration["options"] } = $props();

    let canvas: HTMLCanvasElement;
    let chart: Chart;

    const defaultOptions: ChartConfiguration["options"] | {} = {
        responsive: true,
        maintainAspectRatio: false,
        animation: {
            duration: 1000,
            easing: "easeInOutQuart",
        },
        color: "hsla(210, 21%, 93%, 1)",
        plugins: {
            customChartColors: {
                enabled: true,
            },
            legend: {
                labels: {
                    textAlign: "center",
                    color: "hsla(210, 21%, 93%, 1)",
                    padding: 10,
                    usePointStyle: true,
                    pointStyle: "rectRounded",
                    font: {
                        size: 14,
                    },
                },
                position: "bottom",
            },
        },
    };

    const defaultBarOptions: ChartConfiguration["options"] = {
        scales: {
            x: {
                stacked: true,
                grid: {
                    display: true,
                    color: "hsla(210, 21%, 93%, 0.1)", // Lighter grid lines
                },
                ticks: {
                    color: "hsla(210, 21%, 93%, .75)",
                },
            },
            y: {
                stacked: true,
                grid: {
                    display: true,
                    color: "hsla(210, 21%, 93%, 0.1)", // Lighter grid lines
                },
                ticks: {
                    color: "hsla(210, 21%, 93%, .75)",
                },
            },
        },
    };

    onMount(() => {
        let processedOptions = defaultOptions;
        if (type === "bar") {
            processedOptions = deepMerge(processedOptions, defaultBarOptions);
        }
        processedOptions = deepMerge(processedOptions, options || {});

        chart = new Chart(canvas, {
            type: type,
            data,
            options: processedOptions,
        });
    });

    onDestroy(() => {
        chart?.destroy();
    });

    $effect(() => {
        if (data) {
            log.debug(`Chart data changed, updating chart... (type=${type})`, data);
            chart.data = data;
            chart.update();
        }
    });
</script>

<div class="chart-container">
    <canvas bind:this={canvas} class="chart"></canvas>
</div>

<style>
    .chart-container {
        position: relative;
        width: 100%;
        height: 100%;
    }

    .chart {
        background-color: var(--primary-300); /* Gunmetal background */
        border-radius: 8px; /* Optional: rounded corners */
        padding: 10px; /* Optional: some padding */
    }
</style>
