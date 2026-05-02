export type DashboardCellSpan = 1 | 2 | 3 | 4;

export type DashboardSpan = {
    width: DashboardCellSpan;
    height: DashboardCellSpan;
};

export type DashboardMobileSpan = Partial<DashboardSpan>;

export type DashboardGridPosition = {
    column: number;
    row: number;
};

type DashboardWidgetDefinition<Id extends string = string> = {
    id: Id;
    label: string;
    span: DashboardSpan;
    mobileSpan?: DashboardMobileSpan;
};

export const dashboardWidgetDefinitions = [
    {
        id: "cache-efficiency",
        label: "Cache Efficiency",
        span: { width: 3, height: 3 },
        mobileSpan: { width: 1 },
    },
    { id: "cache-latency", label: "Cache Latency", span: { width: 2, height: 2 } },
    {
        id: "request-latency",
        label: "Request Latency",
        span: { width: 3, height: 3 },
        mobileSpan: { width: 3, height: 2 },
    },
    {
        id: "request-volume",
        label: "Request Volume",
        span: { width: 3, height: 3 },
        mobileSpan: { width: 1 },
    },
    { id: "response-status", label: "Response Status", span: { width: 2, height: 2 } },
    {
        id: "request-coalescing",
        label: "Request Coalescing",
        span: { width: 4, height: 3 },
        mobileSpan: { height: 2 },
    },
    {
        id: "data-transfer",
        label: "Data Transfer",
        span: { width: 2, height: 3 },
        mobileSpan: { height: 3 },
    },
    {
        id: "system-info",
        label: "System Info",
        span: { width: 1, height: 3 },
        mobileSpan: { width: 1 },
    },
    {
        id: "cache-stats",
        label: "Cache Maintenance",
        span: { width: 2, height: 2 },
        mobileSpan: { width: 2, height: 2 },
    },
    {
        id: "cache-storage",
        label: "Cache Storage",
        span: { width: 2, height: 2 },
        mobileSpan: { width: 2, height: 2 },
    },
] as const satisfies readonly DashboardWidgetDefinition[];

export type DashboardWidgetId = (typeof dashboardWidgetDefinitions)[number]["id"];

export type DashboardWidgetLayout = {
    id: DashboardWidgetId;
    span: DashboardSpan;
    mobileSpan?: DashboardMobileSpan;
    position?: DashboardGridPosition;
};

const minSpan = 1;
const maxSpan = 4;

const defaultLayoutsById = new Map(
    dashboardWidgetDefinitions.map((definition) => [definition.id, cloneLayoutItem(definition)]),
);

function isRecord(value: unknown): value is Record<string, unknown> {
    return typeof value === "object" && value !== null && !Array.isArray(value);
}

function cloneMobileSpan(span: DashboardMobileSpan | undefined): DashboardMobileSpan | undefined {
    if (!span) return undefined;

    return { ...span };
}

function cloneLayoutItem(item: DashboardWidgetDefinition | DashboardWidgetLayout) {
    return {
        id: item.id,
        span: { ...item.span },
        ...(item.mobileSpan ? { mobileSpan: cloneMobileSpan(item.mobileSpan) } : {}),
        ...("position" in item && item.position ? { position: { ...item.position } } : {}),
    } as DashboardWidgetLayout;
}

export function isDashboardWidgetId(id: unknown): id is DashboardWidgetId {
    return typeof id === "string" && defaultLayoutsById.has(id as DashboardWidgetId);
}

export function resolveDashboardWidgetId(id: string | undefined): DashboardWidgetId | null {
    return isDashboardWidgetId(id) ? id : null;
}

function clampCellSpan(value: unknown, fallback: DashboardCellSpan): DashboardCellSpan {
    if (typeof value !== "number" || !Number.isFinite(value)) return fallback;

    return Math.min(maxSpan, Math.max(minSpan, Math.round(value))) as DashboardCellSpan;
}

function normalizeSpan(value: unknown, fallback: DashboardSpan): DashboardSpan {
    if (!isRecord(value)) return { ...fallback };

    return {
        width: clampCellSpan(value.width, fallback.width),
        height: clampCellSpan(value.height, fallback.height),
    };
}

function normalizeMobileSpan(
    value: unknown,
    fallback: DashboardMobileSpan | undefined,
): DashboardMobileSpan | undefined {
    if (!isRecord(value)) return cloneMobileSpan(fallback);

    const normalized: DashboardMobileSpan = {};
    if ("width" in value) {
        normalized.width = clampCellSpan(value.width, fallback?.width ?? 1);
    }
    if ("height" in value) {
        normalized.height = clampCellSpan(value.height, fallback?.height ?? 1);
    }

    return Object.keys(normalized).length > 0 ? normalized : cloneMobileSpan(fallback);
}

function normalizeGridPosition(value: unknown): DashboardGridPosition | undefined {
    if (!isRecord(value)) return undefined;
    if (typeof value.column !== "number" || typeof value.row !== "number") return undefined;
    if (!Number.isFinite(value.column) || !Number.isFinite(value.row)) return undefined;

    return {
        column: Math.max(1, Math.round(value.column)),
        row: Math.max(1, Math.round(value.row)),
    };
}

function clampColumns(columns: number) {
    if (!Number.isFinite(columns)) return 1;

    return Math.max(1, Math.round(columns));
}

function effectiveSpanWidth(span: DashboardSpan, columns: number) {
    return Math.min(span.width, columns);
}

function clampWidgetPosition(
    position: DashboardGridPosition,
    span: DashboardSpan,
    columns: number,
) {
    const maxColumn = Math.max(1, columns - effectiveSpanWidth(span, columns) + 1);

    return {
        column: Math.min(maxColumn, Math.max(1, Math.round(position.column))),
        row: Math.max(1, Math.round(position.row)),
    };
}

function occupiedCellKey(column: number, row: number) {
    return `${column}:${row}`;
}

function canPlaceWidget(
    occupiedCells: Set<string>,
    position: DashboardGridPosition,
    span: DashboardSpan,
    columns: number,
) {
    const width = effectiveSpanWidth(span, columns);
    if (position.column < 1 || position.column + width - 1 > columns) return false;

    for (let row = position.row; row < position.row + span.height; row += 1) {
        for (let column = position.column; column < position.column + width; column += 1) {
            if (occupiedCells.has(occupiedCellKey(column, row))) {
                return false;
            }
        }
    }

    return true;
}

function occupyWidgetCells(
    occupiedCells: Set<string>,
    position: DashboardGridPosition,
    span: DashboardSpan,
    columns: number,
) {
    const width = effectiveSpanWidth(span, columns);

    for (let row = position.row; row < position.row + span.height; row += 1) {
        for (let column = position.column; column < position.column + width; column += 1) {
            occupiedCells.add(occupiedCellKey(column, row));
        }
    }
}

function findFirstAvailablePosition(
    occupiedCells: Set<string>,
    span: DashboardSpan,
    columns: number,
) {
    const maxColumn = Math.max(1, columns - effectiveSpanWidth(span, columns) + 1);

    for (let row = 1; ; row += 1) {
        for (let column = 1; column <= maxColumn; column += 1) {
            const position = { column, row };
            if (canPlaceWidget(occupiedCells, position, span, columns)) {
                return position;
            }
        }
    }
}

function compareLayoutPosition(a: DashboardWidgetLayout, b: DashboardWidgetLayout) {
    const rowDelta = (a.position?.row ?? 0) - (b.position?.row ?? 0);
    if (rowDelta !== 0) return rowDelta;

    const columnDelta = (a.position?.column ?? 0) - (b.position?.column ?? 0);
    if (columnDelta !== 0) return columnDelta;

    return a.id.localeCompare(b.id);
}

function packNormalizedDashboardLayout(
    layout: DashboardWidgetLayout[],
    columns: number,
    pinnedWidgetId?: DashboardWidgetId,
) {
    const gridColumns = clampColumns(columns);
    const occupiedCells = new Set<string>();
    const packed: DashboardWidgetLayout[] = [];
    const pinnedWidget = pinnedWidgetId
        ? layout.find((item) => item.id === pinnedWidgetId)
        : undefined;

    if (pinnedWidget?.position) {
        const position = clampWidgetPosition(pinnedWidget.position, pinnedWidget.span, gridColumns);
        packed.push({ ...pinnedWidget, position });
        occupyWidgetCells(occupiedCells, position, pinnedWidget.span, gridColumns);
    }

    for (const item of layout) {
        if (item.id === pinnedWidget?.id) continue;

        const desiredPosition = item.position
            ? clampWidgetPosition(item.position, item.span, gridColumns)
            : undefined;
        const position =
            desiredPosition &&
            canPlaceWidget(occupiedCells, desiredPosition, item.span, gridColumns)
                ? desiredPosition
                : findFirstAvailablePosition(occupiedCells, item.span, gridColumns);

        packed.push({ ...item, position });
        occupyWidgetCells(occupiedCells, position, item.span, gridColumns);
    }

    return packed.sort(compareLayoutPosition);
}

export function defaultDashboardLayout(): DashboardWidgetLayout[] {
    return dashboardWidgetDefinitions.map(cloneLayoutItem);
}

export function normalizeDashboardLayout(value: unknown): DashboardWidgetLayout[] {
    const layout: DashboardWidgetLayout[] = [];
    const seen = new Set<DashboardWidgetId>();

    if (Array.isArray(value)) {
        for (const entry of value) {
            if (!isRecord(entry)) {
                continue;
            }

            if (!isDashboardWidgetId(entry.id) || seen.has(entry.id)) {
                continue;
            }

            const fallback = defaultLayoutsById.get(entry.id);
            if (!fallback) continue;

            const position = normalizeGridPosition(entry.position);
            layout.push({
                id: entry.id,
                span: normalizeSpan(entry.span, fallback.span),
                mobileSpan: normalizeMobileSpan(entry.mobileSpan, fallback.mobileSpan),
                ...(position ? { position } : {}),
            });
            seen.add(entry.id);
        }
    }

    for (const fallback of defaultDashboardLayout()) {
        if (!seen.has(fallback.id)) {
            layout.push(fallback);
        }
    }

    return layout;
}

export function packDashboardLayout(
    layout: DashboardWidgetLayout[],
    columns: number,
): DashboardWidgetLayout[] {
    return packNormalizedDashboardLayout(normalizeDashboardLayout(layout), columns);
}

export function placeDashboardWidget(
    layout: DashboardWidgetLayout[],
    id: DashboardWidgetId,
    position: DashboardGridPosition,
    columns: number,
): DashboardWidgetLayout[] {
    const normalized = normalizeDashboardLayout(layout);
    const widget = normalized.find((item) => item.id === id);

    if (!widget) {
        return packNormalizedDashboardLayout(normalized, columns);
    }

    const gridColumns = clampColumns(columns);
    const positioned = normalized.map((item) =>
        item.id === id
            ? { ...item, position: clampWidgetPosition(position, item.span, gridColumns) }
            : item,
    );

    return packNormalizedDashboardLayout(positioned, gridColumns, id);
}

export function setDashboardWidgetSpan(
    layout: DashboardWidgetLayout[],
    id: DashboardWidgetId,
    span: { width?: number; height?: number },
): DashboardWidgetLayout[] {
    return normalizeDashboardLayout(layout).map((item) => {
        if (item.id !== id) return item;

        return {
            ...item,
            span: {
                width: clampCellSpan(span.width, item.span.width),
                height: clampCellSpan(span.height, item.span.height),
            },
        };
    });
}
