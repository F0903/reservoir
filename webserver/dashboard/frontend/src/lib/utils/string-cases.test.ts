import { describe, it, expect } from "vitest";
import { snakeCaseToCamelCase } from "./string-cases";

describe("string-cases utils", () => {
    describe("snakeCaseToCamelCase", () => {
        it("should convert snake_case to camelCase", () => {
            expect(snakeCaseToCamelCase("hello_world")).toBe("helloWorld");
            expect(snakeCaseToCamelCase("this_is_a_test")).toBe("thisIsATest");
        });

        it("should handle numbers", () => {
            expect(snakeCaseToCamelCase("user_1_id")).toBe("user1Id");
        });

        it("should handle multiple underscores", () => {
            expect(snakeCaseToCamelCase("multiple__underscores")).toBe("multipleUnderscores");
        });

        it("should handle already camelCase or mixed strings", () => {
            expect(snakeCaseToCamelCase("alreadyCamel")).toBe("alreadyCamel");
        });
    });
});
