import { describe, expect, it } from "vitest";
import { dashboardGridPositionFromPoint, type DashboardGridMetrics } from "./dashboard-editor";

const metrics: DashboardGridMetrics = {
    left: 100,
    top: 50,
    columnWidth: 150,
    rowHeight: 150,
    gap: 15,
    columns: 4,
};

describe("dashboard editor grid placement", () => {
    it("maps a dragged widget point to the nearest grid cell", () => {
        expect(
            dashboardGridPositionFromPoint(
                metrics,
                { x: 100 + 165, y: 50 + 165 },
                { width: 1, height: 1 },
            ),
        ).toEqual({ column: 2, row: 2 });
    });

    it("clamps positions before the grid to the first cell", () => {
        expect(
            dashboardGridPositionFromPoint(metrics, { x: 20, y: -40 }, { width: 1, height: 1 }),
        ).toEqual({ column: 1, row: 1 });
    });

    it("clamps wide widgets to the last valid starting column", () => {
        expect(
            dashboardGridPositionFromPoint(metrics, { x: 900, y: 50 }, { width: 3, height: 2 }),
        ).toEqual({ column: 2, row: 1 });
    });
});
