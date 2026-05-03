import { describe, expect, it } from "vitest";
import { userIsAdmin } from "./permissions";

describe("auth permissions", () => {
    it("only treats explicit admin users as admins", () => {
        expect(userIsAdmin({ is_admin: true })).toBe(true);
        expect(userIsAdmin({ is_admin: false })).toBe(false);
        expect(userIsAdmin({})).toBe(false);
        expect(userIsAdmin(null)).toBe(false);
    });
});
