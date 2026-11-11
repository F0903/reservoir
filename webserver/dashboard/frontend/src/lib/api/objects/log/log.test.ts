import { describe, expect, it, vi } from "vitest";
import { getLogTail } from "./log";

vi.mock("$app/navigation", () => ({
    goto: vi.fn(),
}));

vi.mock("$app/paths", () => ({
    resolve: (path: string) => path,
}));

function tailResponse(body: string, contentRange: string) {
    return new Response(body, { status: 206, headers: { "Content-Range": contentRange } });
}

describe("getLogTail", () => {
    it("asks for the end of the log only", async () => {
        const fetchFn = vi.fn().mockResolvedValue(tailResponse("a line\n", "bytes 0-6/7"));

        await getLogTail(fetchFn);

        const headers = new Headers(fetchFn.mock.calls[0][1].headers);
        expect(headers.get("Range")).toMatch(/^bytes=-\d+$/);
    });

    it("drops the cut-off first line of a tail that starts inside the file", async () => {
        const fetchFn = vi
            .fn()
            .mockResolvedValue(tailResponse("d of a line\nfull line\n", "bytes 500-521/522"));

        expect(await getLogTail(fetchFn)).toBe("full line\n");
    });

    it("keeps the first line when the tail is the whole file", async () => {
        const fetchFn = vi
            .fn()
            .mockResolvedValue(tailResponse("first line\nlast line\n", "bytes 0-20/21"));

        expect(await getLogTail(fetchFn)).toBe("first line\nlast line\n");
    });
});
