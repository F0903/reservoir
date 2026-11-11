import { describe, it, expect } from "vitest";
import { formatBytesToLargest } from "./bytestring";

describe("bytestring utils", () => {
    describe("formatBytesToLargest", () => {
        it("should format 0 bytes correctly", () => {
            expect(formatBytesToLargest(0)).toBe("0B");
        });

        it("should format bytes to the largest unit", () => {
            expect(formatBytesToLargest(1024)).toBe("1K");
            expect(formatBytesToLargest(1024 * 1024)).toBe("1M");
            expect(formatBytesToLargest(1024 * 1024 * 1024)).toBe("1G");
        });

        it("should format values below one byte and below zero", () => {
            // Rates such as bytes per second can be fractions.
            expect(formatBytesToLargest(0.5)).toBe("0.5B");
            expect(formatBytesToLargest(-1024)).toBe("-1K");
        });

        it("should respect decimals", () => {
            expect(formatBytesToLargest(1500, 1)).toBe("1.5K");
            expect(formatBytesToLargest(1500, 2)).toBe("1.46K");
        });
    });
});
