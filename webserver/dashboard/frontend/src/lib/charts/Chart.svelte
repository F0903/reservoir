<script lang="ts">
    import Chart from "chart.js/auto";
    import { onDestroy, onMount } from "svelte";
    import type { ChartConfiguration, ChartData, ChartType } from "chart.js";
    import { customChartColors } from "$lib/utils/chart-colors";

    // Register the custom color plugin
    Chart.register(customChartColors);

    let {
        type,
        data,
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
        plugins: {
            customChartColors: {
                enabled: true,
            },
            legend: {
                labels: {
                    textAlign: "left",
                    color: "hsla(210, 21%, 93%, 1)", // Your text color
                },
            },
        },
    };

    onMount(() => {
        chart = new Chart(canvas, {
            type: type,
            data,
            options: {
                ...defaultOptions,
                ...options,
            },
        });
    });

    onDestroy(() => {
        chart?.destroy();
    });
</script>

<canvas bind:this={canvas} class="chart"></canvas>

<style>
    .chart {
        background-color: var(--primary-300); /* Gunmetal background */
        border-radius: 8px; /* Optional: rounded corners */
        padding: 10px; /* Optional: some padding */
    }
</style>
