import { describe, it, expect } from "vitest";
import { patch, type DeepPartial } from "./patch";

describe("patch utility", () => {
    it("should recurse into plain objects", () => {
        const to = { nested: { a: 1, b: 2 } };
        const from = { nested: { b: 3 } };

        const changed = patch(to, from);

        expect(changed).toBe(true);
        expect(to.nested.a).toBe(1);
        expect(to.nested.b).toBe(3);
    });

    it("should handle arrays index-wise by default", () => {
        const to = { list: [1, 2, 3] };
        const from = { list: [1, 4] };

        const changed = patch(to, from);

        expect(changed).toBe(true);
        expect(to.list).toEqual([1, 4]);
        expect(to.list.length).toBe(2);
    });

    it("should handle allowNull option", () => {
        const to = { val: "test" };
        const from = { val: null };

        const changed = patch(to, from as unknown as DeepPartial<typeof to>, { allowNull: false });
        expect(changed).toBe(false);
        expect(to.val).toBe("test");

        const changed2 = patch(to, from as unknown as DeepPartial<typeof to>, { allowNull: true });
        expect(changed2).toBe(true);
        expect(to.val).toBe(null);
    });
});
