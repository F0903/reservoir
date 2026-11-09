import { describe, it, expect, vi, beforeEach } from "vitest";
import { patchConfig } from "./config";
import * as apiHelpers from "#lib/api/api-helpers.ts";

vi.mock("#lib/api/api-helpers.ts", () => ({
    apiPatch: vi.fn(),
    apiGet: vi.fn(),
}));

describe("config api object", () => {
    beforeEach(() => {
        vi.clearAllMocks();
    });

    describe("patchConfig", () => {
        it.each([
            [
                "proxy.listen",
                ":8888",
                {
                    proxy: {
                        listen: ":8888",
                    },
                },
            ],
            [
                "proxy.cache_policy.ignore_cache_control",
                true,
                {
                    proxy: {
                        cache_policy: {
                            ignore_cache_control: true,
                        },
                    },
                },
            ],
        ] as const)(
            "should build a nested patch body for %s",
            async (keyPath, value, expectedBody) => {
                const apiPatchSpy = vi.mocked(apiHelpers.apiPatch);
                apiPatchSpy.mockResolvedValue("success");

                await patchConfig(keyPath, value);

                expect(apiPatchSpy).toHaveBeenCalledWith(
                    "/config",
                    expectedBody,
                    expect.any(Function),
                );
            },
        );
    });
});
