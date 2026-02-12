import { render, screen } from "@testing-library/svelte";
import { describe, it, expect } from "vitest";
import PageTitle from "./PageTitle.svelte";
import { createRawSnippet } from "svelte";

describe("PageTitle component", () => {
    it("should render children", () => {
        // In Svelte 5, we can use createRawSnippet to create a snippet for testing
        // or just pass a component if it's a snippet prop.
        // For simple children, testing library svelte 5 might have a simpler way.

        render(PageTitle, {
            props: {
                children: createRawSnippet(() => ({
                    render: () => "<span>Test Title</span>",
                })),
            },
        });

        const title = screen.getByRole("heading", { level: 1 });
        expect(title).toBeInTheDocument();
        expect(title).toHaveTextContent("Test Title");
        expect(title).toHaveClass("page-title");
    });
});
