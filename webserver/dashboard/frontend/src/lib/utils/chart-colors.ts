// https://karlynelson.com/posts/chartjs-custom-color-palette-responsive/

import { Chart, type ChartDataset } from "chart.js";
import { DoughnutController, PolarAreaController } from "chart.js";

Chart.register(DoughnutController, PolarAreaController);

const COLORS = [
    "hsla(188, 34%, 43%)",
    "hsla(188, 34%, 30%)",
    "hsla(22, 70%, 44%)",
    "hsla(22, 70%, 64%)",
];

function getBorderColor(i: number) {
    return COLORS[i % COLORS.length];
}

function getBackgroundColor(i: number) {
    return COLORS[i % COLORS.length];
}

function colorizeDefaultDataset(dataset: ChartDataset, i: number) {
    dataset.borderColor = getBorderColor(i);
    dataset.backgroundColor = getBackgroundColor(i);
    return ++i;
}

function colorizeDoughnutDataset(dataset: ChartDataset, i: number) {
    dataset.backgroundColor = dataset.data.map(() => getBorderColor(i++));
    return i;
}

function colorizePolarAreaDataset(dataset: ChartDataset, i: number) {
    dataset.backgroundColor = dataset.data.map(() => getBackgroundColor(i++));
    return i;
}

function getColorizer(chart: Chart) {
    let i = 0;
    return (dataset: ChartDataset, datasetIndex: number) => {
        const controller = chart.getDatasetMeta(datasetIndex).controller;
        if (controller instanceof DoughnutController) {
            i = colorizeDoughnutDataset(dataset, i);
        } else if (controller instanceof PolarAreaController) {
            i = colorizePolarAreaDataset(dataset, i);
        } else if (controller) {
            i = colorizeDefaultDataset(dataset, i);
        }
    };
}

export const customChartColors = {
    id: "customChartColors",
    defaults: {
        enabled: true,
    },
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    beforeLayout(chart: Chart, _args: any, options: any) {
        if (!options.enabled) {
            return;
        }
        const {
            data: { datasets },
        } = chart.config;
        const colorizer = getColorizer(chart);
        datasets.forEach(colorizer);
    },
};
