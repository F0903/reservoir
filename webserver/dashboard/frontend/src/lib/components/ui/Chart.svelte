<script lang="ts">
    import Chart from "chart.js/auto";
    import { onMount } from "svelte";
    import type { ChartConfiguration, ChartData, ChartType } from "chart.js";
    import { log } from "$lib/utils/logger";
    import { patch } from "$lib/utils/patch";

    let {
        type,
        data = $bindable(),
        options = {},
    }: { type: ChartType; data: ChartData; options?: ChartConfiguration["options"] } = $props();

    const css = getComputedStyle(document.documentElement);

    // Helper to resolve var(--color) to actual color string
    function resolveColor(color: string | string[] | undefined): string | string[] | undefined {
        if (!color) return color;
        if (Array.isArray(color)) return color.map((c) => resolveColor(c) as string);
        if (typeof color !== "string") return color;

        if (color.startsWith("var(")) {
            const varName = color.slice(4, -1).trim();
            const resolved = css.getPropertyValue(varName).trim();
            return resolved || color;
        }
        return color;
    }

    function resolveDatasetColors(chartData: ChartData) {
        if (!chartData.datasets) return;
        chartData.datasets.forEach((ds) => {
            if (ds.backgroundColor) {
                const resolved = resolveColor(ds.backgroundColor as string | string[]);
                // We cast back to the internal expected type which can be many things, but string|string[] covers our usage.
                ds.backgroundColor = resolved as typeof ds.backgroundColor;
            }
            if (ds.borderColor) {
                const resolved = resolveColor(ds.borderColor as string | string[]);
                ds.borderColor = resolved as typeof ds.borderColor;
            }
        });
    }

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
            duration: 800,
            easing: "easeOutQuart",
        },
        color: "rgba(255, 255, 255, 0.7)",
        elements: {
            arc: {
                borderWidth: 0,
                hoverBorderWidth: 0,
            },
            bar: {
                borderRadius: 4,
                borderWidth: 0,
                hoverBorderWidth: 0,
                borderSkipped: false,
            },
        },
        plugins: {
            legend: {
                labels: {
                    textAlign: "center",
                    color: "rgba(255, 255, 255, 0.6)",
                    padding: 15,
                    usePointStyle: true,
                    pointStyle: "circle",
                    font: {
                        weight: "bold",
                        size: 11,
                    },
                },
                position: "bottom",
            },
            tooltip: {
                backgroundColor: "rgba(10, 10, 10, 0.9)",
                titleColor: "var(--secondary-300)",
                bodyColor: "#fff",
                padding: 10,
                cornerRadius: 8,
                displayColors: true,
            },
        },
    };

    const defaultDonutOptions: ChartConfiguration["options"] = {};

    const defaultBarOptions: ChartConfiguration["options"] = {
        scales: {
            x: {
                stacked: true,
                grid: {
                    display: false, // Cleaner X-axis
                },
                ticks: {
                    color: "rgba(255, 255, 255, 0.5)",
                    font: { size: 10 },
                },
            },
            y: {
                stacked: true,
                grid: {
                    display: true,
                    color: "rgba(255, 255, 255, 0.05)", // Very subtle grid
                    drawTicks: false,
                },
                ticks: {
                    color: "rgba(255, 255, 255, 0.5)",
                    font: { size: 10 },
                    padding: 8,
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

        resolveDatasetColors(data);

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
        if (!data || !chart) return;

        log.debug(`Chart data changed, updating chart... (type=${type})`, data);
        resolveDatasetColors(data);
        chart.data = data;
        chart.update("none"); // Use 'none' for instant updates during rapid data flow
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
        min-height: 0;
    }

    .chart {
        display: block;
        width: 100% !important;
        height: 100% !important;
    }
</style>
