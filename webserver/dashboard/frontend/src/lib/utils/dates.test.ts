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
        it("should format seconds", () => {
            const now = new Date("2024-01-01T12:00:00Z");
            vi.setSystemTime(now);

            const start = new Date(now.getTime() - 30 * 1000);
            expect(formatTimeSinceDate(start)).toBe("30s");
        });

        it("should format minutes and seconds", () => {
            const now = new Date("2024-01-01T12:00:00Z");
            vi.setSystemTime(now);

            const start = new Date(now.getTime() - (5 * 60 * 1000 + 30 * 1000));
            expect(formatTimeSinceDate(start)).toBe("5m 30s");
        });

        it("should format hours, minutes and seconds", () => {
            const now = new Date("2024-01-01T12:00:00Z");
            vi.setSystemTime(now);

            const start = new Date(now.getTime() - (2 * 3600 * 1000 + 5 * 60 * 1000 + 30 * 1000));
            expect(formatTimeSinceDate(start)).toBe("2h 5m 30s");
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
