import { describe, expect, it, vi } from "vitest";
import { clearCache } from "./cache";

describe("cache api", () => {
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
