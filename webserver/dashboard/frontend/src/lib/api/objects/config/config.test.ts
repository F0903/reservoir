import { describe, it, expect, vi, beforeEach } from "vitest";
import { patchConfig } from "./config";
import * as apiHelpers from "$lib/api/api-helpers";

vi.mock("$lib/api/api-helpers", () => ({
    apiPatch: vi.fn(),
    apiGet: vi.fn(),
}));

describe("config api object", () => {
    beforeEach(() => {
        vi.clearAllMocks();
    });

    describe("patchConfig", () => {
        it("should correctly build a nested object for a deep property", async () => {
            const apiPatchSpy = vi.mocked(apiHelpers.apiPatch);
            apiPatchSpy.mockResolvedValue("success");

            await patchConfig("proxy.listen", ":8888");

            expect(apiPatchSpy).toHaveBeenCalledWith(
                "/config",
                {
                    proxy: {
                        listen: ":8888",
                    },
                },
                expect.any(Function),
            );
        });

        it("should correctly build a nested object for another sub-config", async () => {
            const apiPatchSpy = vi.mocked(apiHelpers.apiPatch);
            apiPatchSpy.mockResolvedValue("success");

            await patchConfig("cache.max_cache_size", 5000);

            expect(apiPatchSpy).toHaveBeenCalledWith(
                "/config",
                {
                    cache: {
                        max_cache_size: 5000,
                    },
                },
                expect.any(Function),
            );
        });

        it("should correctly build a deeply nested object for logging", async () => {
            const apiPatchSpy = vi.mocked(apiHelpers.apiPatch);
            apiPatchSpy.mockResolvedValue("success");

            await patchConfig("logging.level", "DEBUG");

            expect(apiPatchSpy).toHaveBeenCalledWith(
                "/config",
                {
                    logging: {
                        level: "DEBUG",
                    },
                },
                expect.any(Function),
            );
        });

        it("should correctly build a deeply nested object for cache specific properties", async () => {
            const apiPatchSpy = vi.mocked(apiHelpers.apiPatch);
            apiPatchSpy.mockResolvedValue("success");

            await patchConfig("cache.file.dir", "/new/path");

            expect(apiPatchSpy).toHaveBeenCalledWith(
                "/config",
                {
                    cache: {
                        file: {
                            dir: "/new/path",
                        },
                    },
                },
                expect.any(Function),
            );
        });
    });
});
