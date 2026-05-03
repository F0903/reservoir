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

function smallPositionedLayout(): DashboardWidgetLayout[] {
    return defaultDashboardLayout().map((item, index) => ({
        ...item,
        span: { width: 2, height: 2 },
        position: { column: 1, row: 20 + index * 3 },
    })) as DashboardWidgetLayout[];
}

describe("dashboard layout", () => {
    it("returns a complete default layout", () => {
        const layout = defaultDashboardLayout();

        expect(layout.map((item) => item.id)).toEqual([
            "cache-efficiency",
            "cache-latency",
            "request-latency",
            "request-volume",
            "response-status",
            "request-coalescing",
            "data-transfer",
            "process-info",
            "cache-maintenance",
            "cache-storage",
        ]);
        expect(layout[0].span).toEqual({ width: 6, height: 6 });
    });

    it("normalizes saved order and clamps saved spans and positions", () => {
        const layout = normalizeDashboardLayout([
            {
                id: "cache-storage",
                span: { width: 12, height: -1 },
                mobileSpan: { width: 12 },
                position: { column: -4, row: 2.6 },
            },
            { id: "cache-efficiency", span: { width: 2, height: 2 } },
            { id: "unknown-widget", span: { width: 4, height: 4 } },
            { id: "cache-storage", span: { width: 1, height: 1 } },
        ]);

        expect(layout[0]).toEqual({
            id: "cache-storage",
            span: { width: 8, height: 1 },
            mobileSpan: { width: 8 },
            position: { column: 1, row: 3 },
        });
        expect(layout[1].id).toBe("cache-efficiency");
        expect(layout.map((item) => item.id)).toHaveLength(defaultDashboardLayout().length);
        expect(layout.filter((item) => item.id === "cache-storage")).toHaveLength(1);
    });

    it("appends missing widgets after saved widgets", () => {
        const layout = normalizeDashboardLayout([
            { id: "process-info", span: { width: 4, height: 4 } },
        ]);

        expect(layout[0].id).toBe("process-info");
        expect(layout.at(-1)?.id).toBe("cache-storage");
    });

    it("sets widget spans within supported grid bounds", () => {
        const layout = setDashboardWidgetSpan(defaultDashboardLayout(), "process-info", {
            width: 6,
            height: 16,
        });

        expect(layout.find((item) => item.id === "process-info")?.span).toEqual({
            width: 6,
            height: 8,
        });
    });

    it("packs dashboard widgets into explicit non-overlapping grid positions", () => {
        const layout = packDashboardLayout(defaultDashboardLayout().slice(0, 4), 8);

        expect(layout.every((widget) => widget.position)).toBe(true);
        expect(layout[0].position).toEqual({ column: 1, row: 1 });
        occupiedCells(layout, 8);
    });

    it("moves only the widget occupying the requested grid cell", () => {
        const initialLayout = smallPositionedLayout().map((item) => {
            if (item.id === "cache-efficiency") return { ...item, position: { column: 1, row: 1 } };
            if (item.id === "cache-latency") return { ...item, position: { column: 3, row: 1 } };
            if (item.id === "request-latency") return { ...item, position: { column: 5, row: 1 } };

            return item;
        });

        const layout = placeDashboardWidget(
            initialLayout,
            "cache-efficiency",
            { column: 3, row: 1 },
            8,
        );

        expect(layout.find((widget) => widget.id === "cache-efficiency")?.position).toEqual({
            column: 3,
            row: 1,
        });
        expect(layout.find((widget) => widget.id === "cache-latency")?.position).toEqual({
            column: 1,
            row: 1,
        });
        expect(layout.find((widget) => widget.id === "request-latency")?.position).toEqual({
            column: 5,
            row: 1,
        });
        occupiedCells(layout, 8);
    });

    it("does not move widgets when a dragged widget would require cascading displacement", () => {
        const initialLayout = smallPositionedLayout().map((item) => {
            if (item.id === "cache-efficiency") {
                return { ...item, span: { width: 4, height: 2 }, position: { column: 1, row: 10 } };
            }
            if (item.id === "cache-latency") return { ...item, position: { column: 1, row: 1 } };
            if (item.id === "request-latency") return { ...item, position: { column: 3, row: 1 } };

            return item;
        }) as DashboardWidgetLayout[];

        const layout = placeDashboardWidget(
            initialLayout,
            "cache-efficiency",
            { column: 1, row: 1 },
            8,
        );

        expect(layout.find((widget) => widget.id === "cache-efficiency")?.position).toEqual({
            column: 1,
            row: 10,
        });
        expect(layout.find((widget) => widget.id === "cache-latency")?.position).toEqual({
            column: 1,
            row: 1,
        });
        expect(layout.find((widget) => widget.id === "request-latency")?.position).toEqual({
            column: 3,
            row: 1,
        });
        occupiedCells(layout, 8);
    });

    it("clamps placed widgets to the available column range", () => {
        const initialLayout = smallPositionedLayout().map((item) =>
            item.id === "cache-efficiency"
                ? { ...item, span: { width: 6, height: 2 }, position: { column: 1, row: 1 } }
                : item,
        ) as DashboardWidgetLayout[];

        const layout = placeDashboardWidget(
            initialLayout,
            "cache-efficiency",
            { column: 16, row: 2 },
            8,
        );

        expect(layout.find((widget) => widget.id === "cache-efficiency")?.position).toEqual({
            column: 3,
            row: 2,
        });
        occupiedCells(layout, 8);
    });
});
