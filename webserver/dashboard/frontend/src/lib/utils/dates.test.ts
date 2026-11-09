import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { formatTimeSinceDate } from "./dates";

describe("dates utils", () => {
    beforeEach(() => {
        vi.useFakeTimers();
    });

    afterEach(() => {
        vi.useRealTimers();
    });

    describe("formatTimeSinceDate", () => {
        it.each([
            ["30s", 30 * 1000],
            ["5m 30s", 5 * 60 * 1000 + 30 * 1000],
            ["2h 5m 30s", 2 * 3600 * 1000 + 5 * 60 * 1000 + 30 * 1000],
        ] as const)("should format elapsed time as %s", (expected, elapsedMs) => {
            const now = new Date("2024-01-01T12:00:00Z");
            vi.setSystemTime(now);

            const start = new Date(now.getTime() - elapsedMs);
            expect(formatTimeSinceDate(start)).toBe(expected);
        });

        it("should limit to 3 parts", () => {
            const now = new Date("2024-01-01T12:00:00Z");
            vi.setSystemTime(now);

            // 1 day, 2 hours, 3 minutes, 4 seconds
            const start = new Date(
                now.getTime() - (24 * 3600 * 1000 + 2 * 3600 * 1000 + 3 * 60 * 1000 + 4 * 1000),
            );
            expect(formatTimeSinceDate(start)).toBe("1d 2h 3m");
        });
    });
});
