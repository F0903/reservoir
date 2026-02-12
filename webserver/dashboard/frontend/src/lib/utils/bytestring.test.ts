import { describe, it, expect } from "vitest";
import { formatBytesToLargest, formatBytes, parseByteString } from "./bytestring";

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

        it("should respect decimals", () => {
            expect(formatBytesToLargest(1500, 1)).toBe("1.5K");
            expect(formatBytesToLargest(1500, 2)).toBe("1.46K");
        });
    });

    describe("formatBytes", () => {
        it("should format 0 bytes with specified unit", () => {
            expect(formatBytes(0, "K")).toBe("0B"); // Wait, the code says unitLabels[0] which is 'B'
        });

        it("should format bytes to a fixed unit", () => {
            expect(formatBytes(1024, "K")).toBe("1K");
            expect(formatBytes(1024 * 1024, "K")).toBe("1024K");
            expect(formatBytes(1024, "B")).toBe("1024B");
        });
    });

    describe("parseByteString", () => {
        it("should parse valid byte strings", () => {
            expect(parseByteString("1B")).toBe(1);
            expect(parseByteString("1K")).toBe(1024);
            expect(parseByteString("1M")).toBe(1024 * 1024);
            expect(parseByteString("1G")).toBe(1024 * 1024 * 1024);
        });

        it("should be case insensitive", () => {
            expect(parseByteString("1k")).toBe(1024);
            expect(parseByteString("1m")).toBe(1024 * 1024);
        });

        it("should throw error for invalid strings", () => {
            expect(() => parseByteString("invalid")).toThrow("Invalid byte string: invalid");
            expect(() => parseByteString("1X")).toThrow();
        });
    });
});
