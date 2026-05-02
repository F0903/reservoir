import { describe, it, expect, vi, beforeEach } from "vitest";
import { goto } from "$app/navigation";
import { apiDelete, apiGet, apiGetTextStream, apiPatch, apiPost } from "./api-helpers";
import UnauthorizedError from "./unauthorized-error";

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
                text: () => Promise.resolve(""),
            });

            await expect(apiGet("/test", fetchFn)).rejects.toThrow(
                "Failed to fetch from 'http://localhost/api/test': 500 Internal Server Error",
            );
        });

        it("should include response body on non-ok response when present", async () => {
            const fetchFn = vi.fn().mockResolvedValue({
                status: 400,
                ok: false,
                statusText: "Bad Request",
                url: "http://localhost/api/test",
                text: () => Promise.resolve("cache.max_cache_size must be greater than 0\n"),
            });

            await expect(apiGet("/test", fetchFn)).rejects.toThrow(
                "Failed to fetch from 'http://localhost/api/test': 400 Bad Request: cache.max_cache_size must be greater than 0",
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

    describe("apiGetTextStream", () => {
        it("should not redirect on 401 when redirect is disabled", async () => {
            const fetchFn = vi.fn().mockResolvedValue({
                status: 401,
                ok: false,
            });

            await expect(apiGetTextStream("/stream", fetchFn, null)).rejects.toThrow(
                UnauthorizedError,
            );
            expect(goto).not.toHaveBeenCalled();
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

    describe("apiPatch", () => {
        it("should patch data successfully", async () => {
            const fetchFn = vi.fn().mockResolvedValue({
                status: 200,
                ok: true,
                headers: {
                    get: () => "text/plain",
                },
                text: () => Promise.resolve("restart required"),
            });

            const payload = { key: "value" };
            const result = await apiPatch<string>("/test", payload, fetchFn);

            expect(fetchFn).toHaveBeenCalledWith(
                "/api/test",
                expect.objectContaining({
                    method: "PATCH",
                    body: JSON.stringify(payload),
                }),
            );
            expect(result).toEqual("restart required");
        });
    });

    describe("apiDelete", () => {
        it("should delete successfully", async () => {
            const fetchFn = vi.fn().mockResolvedValue({
                status: 204,
                ok: true,
                headers: {
                    get: () => null,
                },
                text: () => Promise.resolve(""),
            });

            await apiDelete<void>("/test", fetchFn);

            expect(fetchFn).toHaveBeenCalledWith(
                "/api/test",
                expect.objectContaining({
                    method: "DELETE",
                    credentials: "same-origin",
                }),
            );
        });
    });
});
