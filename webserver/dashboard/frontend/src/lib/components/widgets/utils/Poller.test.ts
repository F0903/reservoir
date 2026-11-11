import { render } from "@testing-library/svelte";
import { createRawSnippet } from "svelte";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import Poller from "./Poller.svelte";

// jsdom has no IntersectionObserver. The Poller starts out visible, which is all this test needs.
vi.mock("#lib/attachments/visible.ts", () => ({
    visible: () => () => {},
}));

vi.mock("#lib/utils/logger.ts", () => ({
    log: {
        debug: vi.fn(),
    },
}));

const children = createRawSnippet(() => ({
    render: () => "<div>Polled content</div>",
}));

describe("Poller", () => {
    beforeEach(() => {
        vi.useFakeTimers();
    });

    afterEach(() => {
        vi.useRealTimers();
    });

    it("stops polling when it is removed while visible", () => {
        const pollFn = vi.fn();
        const { unmount } = render(Poller, { props: { pollFn, pollInterval: 1000, children } });

        vi.advanceTimersByTime(1000);
        const callsWhileMounted = pollFn.mock.calls.length;
        expect(callsWhileMounted).toBeGreaterThan(0);

        unmount();
        vi.advanceTimersByTime(5000);

        expect(pollFn).toHaveBeenCalledTimes(callsWhileMounted);
    });
});
