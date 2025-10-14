<script lang="ts">
    import Chart from "chart.js/auto";
    import { onDestroy, onMount } from "svelte";
    import type { ChartConfiguration, ChartData, ChartDataset, ChartType } from "chart.js";
    import { customChartColors } from "$lib/utils/chart-colors";
    import { log } from "$lib/utils/logger";
    import { patch } from "$lib/utils/patch";

    // Register the custom color plugin
    Chart.register(customChartColors);

    let {
        type,
        data = $bindable(),
        options = {},
    }: { type: ChartType; data: ChartData; options?: ChartConfiguration["options"] } = $props();

    const css = getComputedStyle(document.documentElement);

    const defaultOptions: ChartConfiguration["options"] & {
        plugins?: {
            customChartColors?: {
                enabled?: boolean;
            };
        };
    } = {
        responsive: true,
        maintainAspectRatio: false,
        animation: {
            duration: 1000,
            easing: "easeInOutQuart",
        },
        color: css.getPropertyValue("--secondary-600"),
        elements: {
            arc: {
                borderWidth: 1.5,
                borderColor: css.getPropertyValue("--text-400"),
                hoverBorderWidth: 0,
            },
            bar: {
                borderWidth: 1.5,
                borderColor: css.getPropertyValue("--text-400"),
                hoverBorderWidth: 0,
                borderSkipped: "start",
            },
        },
        plugins: {
            customChartColors: {
                enabled: true,
            },
            legend: {
                labels: {
                    textAlign: "center",
                    color: css.getPropertyValue("--text-400"),
                    padding: 10,
                    usePointStyle: true,
                    pointStyle: "rectRounded",
                    font: {
                        weight: 550,
                        size: 14,
                    },
                },
                position: "bottom",
            },
        },
    };

    const defaultDonutOptions: ChartConfiguration["options"] = {};

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

    let canvas: HTMLCanvasElement;
    let chart: Chart;

    onMount(() => {
        let processedOptions = defaultOptions;
        switch (type) {
            case "bar":
                patch(processedOptions, defaultBarOptions);
                break;
            case "doughnut":
                patch(processedOptions, defaultDonutOptions);
                break;

            default:
                break;
        }
        patch(processedOptions, options || {});

        chart = new Chart(canvas, {
            type: type,
            data,
            options: processedOptions,
        });

        return () => {
            chart.destroy();
        };
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

        display: block;
        width: 100% !important; /* For some reason, it will only automatically resize with !important set. */
    }
</style>
