import { isRedirect } from "@sveltejs/kit";
import { describe, expect, it, vi } from "vitest";
import { load } from "./+page";

vi.mock("$app/env", () => ({
    browser: true,
}));

describe("login page load", () => {
    it("redirects to bootstrap when setup is still required", async () => {
        const fetchFn = vi.fn().mockResolvedValue({
            ok: true,
            json: () => Promise.resolve({ bootstrap_required: true }),
        });

        try {
            await load(loadEvent(fetchFn));
            throw new Error("expected login load to redirect");
        } catch (err) {
            expect(isRedirect(err)).toBe(true);
            expect(err).toMatchObject({ status: 302, location: "/bootstrap" });
        }

        expect(fetchFn).toHaveBeenCalledWith("/api/auth/bootstrap", {
            credentials: "same-origin",
        });
    });

    it("returns login data when bootstrap is not required", async () => {
        const fetchFn = vi.fn().mockResolvedValue({
            ok: true,
            json: () => Promise.resolve({ bootstrap_required: false }),
        });

        await expect(
            load(loadEvent(fetchFn, "http://localhost/login?return=%2Fdashboard")),
        ).resolves.toEqual({
            return: "/dashboard",
        });
    });
});

function loadEvent(fetchFn: typeof fetch, rawUrl = "http://localhost/login") {
    return {
        fetch: fetchFn,
        url: new URL(rawUrl),
    } as Parameters<typeof load>[0];
}
