import { describe, it, expect, vi, beforeEach } from "vitest";
import { apiGet, apiPost, UnauthorizedError } from "./api-helpers";

// Mock $app/navigation and $app/paths
vi.mock("$app/navigation", () => ({
    goto: vi.fn(),
}));

vi.mock("$app/paths", () => ({
    resolve: (path: string) => path,
}));

describe("api-helpers", () => {
    beforeEach(() => {
        vi.clearAllMocks();
    });

    describe("apiGet", () => {
        it("should fetch data successfully", async () => {
            const mockData = { foo: "bar" };
            const fetchFn = vi.fn().mockResolvedValue({
                status: 200,
                ok: true,
                json: () => Promise.resolve(mockData),
            });

            const result = await apiGet("/test", fetchFn);

            expect(fetchFn).toHaveBeenCalledWith(expect.anything(), expect.any(Object));
            const calledUrl = fetchFn.mock.calls[0][0].toString();
            expect(calledUrl).toContain("/api/test");
            expect(result).toEqual(mockData);
        });

        it("should throw Error on non-ok response", async () => {
            const fetchFn = vi.fn().mockResolvedValue({
                status: 500,
                ok: false,
                statusText: "Internal Server Error",
                url: "http://localhost/api/test",
            });

            await expect(apiGet("/test", fetchFn)).rejects.toThrow(
                "Failed to fetch from 'http://localhost/api/test': 500 Internal Server Error",
            );
        });

        it("should throw UnauthorizedError on 401", async () => {
            const fetchFn = vi.fn().mockResolvedValue({
                status: 401,
                ok: false,
            });

            await expect(apiGet("/test", fetchFn)).rejects.toThrow(UnauthorizedError);
        });
    });

    describe("apiPost", () => {
        it("should post data successfully", async () => {
            const mockData = { success: true };
            const fetchFn = vi.fn().mockResolvedValue({
                status: 200,
                ok: true,
                headers: {
                    get: () => "application/json",
                },
                json: () => Promise.resolve(mockData),
            });

            const payload = { key: "value" };
            const result = await apiPost("/test", payload, fetchFn);

            expect(fetchFn).toHaveBeenCalledWith(
                "/api/test",
                expect.objectContaining({
                    method: "POST",
                    body: JSON.stringify(payload),
                }),
            );
            expect(result).toEqual(mockData);
        });
    });
});
