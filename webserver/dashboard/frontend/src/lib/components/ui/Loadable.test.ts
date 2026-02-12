import { render, screen } from "@testing-library/svelte";
import { describe, it, expect } from "vitest";
import Loadable from "./Loadable.svelte";
import { createRawSnippet, type Snippet } from "svelte";

describe("Loadable component", () => {
    // We can use unknown instead of any to be type-safe.
    // We cast to Snippet<[never]> because that's the "bottom" type for snippet arguments,
    // making it compatible with most inferred types in these tests.
    const childrenSnippet = createRawSnippet((data: () => unknown) => ({
        render: () => `<div>Loaded: ${data()}</div>`,
    })) as unknown as Snippet<[data: never]>;

    it("should render loading box when state is null", () => {
        const { container } = render(Loadable<string>, {
            props: {
                state: null,
                children: childrenSnippet as unknown as Snippet<[data: string]>,
            },
        });

        expect(container.querySelector(".loading-box")).toBeInTheDocument();
        expect(screen.queryByText(/Loaded:/)).not.toBeInTheDocument();
    });

    it("should render loading box when state is undefined", () => {
        const { container } = render(Loadable<string>, {
            props: {
                state: undefined,
                children: childrenSnippet as unknown as Snippet<[data: string]>,
            },
        });

        expect(container.querySelector(".loading-box")).toBeInTheDocument();
    });

    it("should render error when error prop is provided", () => {
        type TestData = { some: string };
        render(Loadable<TestData>, {
            props: {
                state: { some: "data" },
                error: "Failed to load",
                children: childrenSnippet as unknown as Snippet<[data: TestData]>,
            },
        });

        expect(screen.getByText("Error!")).toBeInTheDocument();
        expect(screen.getByText("Failed to load")).toBeInTheDocument();
        expect(screen.queryByText(/Loaded:/)).not.toBeInTheDocument();
    });

    it("should render children when state is provided and no error", () => {
        render(Loadable<string>, {
            props: {
                state: "test-data",
                children: childrenSnippet as unknown as Snippet<[data: string]>,
            },
        });

        expect(screen.getByText("Loaded: test-data")).toBeInTheDocument();
        expect(screen.queryByText("Error!")).not.toBeInTheDocument();
        expect(screen.queryByRole("loading-box")).not.toBeInTheDocument();
    });
});
