import { describe, expect, it, vi } from "vitest";
import { clearCache, getCacheStatus } from "./cache";

describe("cache api", () => {
    it("fetches cache status", async () => {
        const status = {
            type: "memory",
            entries: 2,
            bytes: 512,
            max_bytes: 1024,
            memory_cap_bytes: 2048,
        };
        const fetchFn = vi.fn().mockResolvedValue({
            status: 200,
            ok: true,
            json: () => Promise.resolve(status),
        });

        const result = await getCacheStatus(fetchFn);

        expect(fetchFn).toHaveBeenCalledWith(
            expect.anything(),
            expect.objectContaining({ method: "GET" }),
        );
        expect(fetchFn.mock.calls[0][0].toString()).toContain("/api/cache/status");
        expect(result).toEqual(status);
    });

    it("clears cache", async () => {
        const fetchFn = vi.fn().mockResolvedValue({
            status: 204,
            ok: true,
            headers: {
                get: () => null,
            },
            text: () => Promise.resolve(""),
        });

        await clearCache(fetchFn);

        expect(fetchFn).toHaveBeenCalledWith(
            "/api/cache/clear",
            expect.objectContaining({
                method: "POST",
                body: "{}",
            }),
        );
    });
});
