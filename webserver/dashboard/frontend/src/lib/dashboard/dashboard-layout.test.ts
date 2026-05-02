import { describe, expect, it } from "vitest";
import {
    defaultDashboardLayout,
    normalizeDashboardLayout,
    packDashboardLayout,
    placeDashboardWidget,
    setDashboardWidgetSpan,
    type DashboardWidgetLayout,
} from "./dashboard-layout";

function occupiedCells(layout: DashboardWidgetLayout[], columns: number) {
    const cells = new Set<string>();

    for (const widget of layout) {
        if (!widget.position) continue;

        const width = Math.min(widget.span.width, columns);
        for (let row = widget.position.row; row < widget.position.row + widget.span.height; row++) {
            for (
                let column = widget.position.column;
                column < widget.position.column + width;
                column++
            ) {
                const key = `${column}:${row}`;
                expect(cells.has(key)).toBe(false);
                cells.add(key);
            }
        }
    }

    return cells;
}

describe("dashboard layout", () => {
    it("returns a complete default layout", () => {
        const layout = defaultDashboardLayout();

        expect(layout.map((item) => item.id)).toEqual([
            "cache-efficiency",
            "cache-activity",
            "cache-latency",
            "request-latency",
            "request-volume",
            "response-status",
            "request-coalescing",
            "data-transfer",
            "system-info",
            "cache-stats",
            "cache-storage",
        ]);
        expect(layout[0].span).toEqual({ width: 3, height: 3 });
    });

    it("normalizes saved order and clamps saved spans and positions", () => {
        const layout = normalizeDashboardLayout([
            {
                id: "cache-storage",
                span: { width: 9, height: -1 },
                mobileSpan: { width: 12 },
                position: { column: -4, row: 2.6 },
            },
            { id: "cache-efficiency", span: { width: 2, height: 2 } },
            { id: "unknown-widget", span: { width: 4, height: 4 } },
            { id: "cache-storage", span: { width: 1, height: 1 } },
        ]);

        expect(layout[0]).toEqual({
            id: "cache-storage",
            span: { width: 4, height: 1 },
            mobileSpan: { width: 4 },
            position: { column: 1, row: 3 },
        });
        expect(layout[1].id).toBe("cache-efficiency");
        expect(layout.map((item) => item.id)).toHaveLength(defaultDashboardLayout().length);
        expect(layout.filter((item) => item.id === "cache-storage")).toHaveLength(1);
    });

    it("appends missing widgets after saved widgets", () => {
        const layout = normalizeDashboardLayout([
            { id: "system-info", span: { width: 2, height: 2 } },
        ]);

        expect(layout[0].id).toBe("system-info");
        expect(layout.at(-1)?.id).toBe("cache-storage");
        expect(layout.some((item) => item.id === "cache-activity")).toBe(true);
    });

    it("sets widget spans within supported grid bounds", () => {
        const layout = setDashboardWidgetSpan(defaultDashboardLayout(), "system-info", {
            width: 3,
            height: 12,
        });

        expect(layout.find((item) => item.id === "system-info")?.span).toEqual({
            width: 3,
            height: 4,
        });
    });

    it("packs dashboard widgets into explicit non-overlapping grid positions", () => {
        const layout = packDashboardLayout(defaultDashboardLayout().slice(0, 4), 4);

        expect(layout.every((widget) => widget.position)).toBe(true);
        expect(layout[0].position).toEqual({ column: 1, row: 1 });
        occupiedCells(layout, 4);
    });

    it("places a dragged widget at the requested grid cell and packs other widgets around it", () => {
        const layout = placeDashboardWidget(
            defaultDashboardLayout().slice(0, 5),
            "response-status",
            { column: 2, row: 2 },
            4,
        );

        expect(layout.find((widget) => widget.id === "response-status")?.position).toEqual({
            column: 2,
            row: 2,
        });
        occupiedCells(layout, 4);
    });

    it("clamps placed widgets to the available column range", () => {
        const layout = placeDashboardWidget(
            defaultDashboardLayout().slice(0, 3),
            "cache-efficiency",
            { column: 8, row: 2 },
            4,
        );

        expect(layout.find((widget) => widget.id === "cache-efficiency")?.position).toEqual({
            column: 2,
            row: 2,
        });
        occupiedCells(layout, 4);
    });
});
