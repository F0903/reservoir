<script lang="ts">
    import type { Snippet } from "svelte";

    let {
        text,
        align = "center",
        children,
    }: {
        text: string;
        align?: "start" | "center" | "end";
        children: Snippet;
    } = $props();
</script>

<span
    class="tooltip"
    class:align-start={align === "start"}
    class:align-end={align === "end"}
    data-tooltip={text}
>
    {@render children()}
</span>

<style>
    .tooltip {
        position: relative;
        display: inline-grid;
        place-items: center;
    }

    .tooltip::before,
    .tooltip::after {
        position: absolute;
        top: calc(100% + 0.45rem);
        left: 50%;
        z-index: 100;
        pointer-events: none;
        opacity: 0;
        transition:
            opacity 120ms ease,
            transform 120ms ease;
    }

    .tooltip::before {
        content: "";
        width: 0.45rem;
        height: 0.45rem;
        background-color: var(--primary-700);
        border-left: 1px solid rgba(255, 255, 255, 0.08);
        border-top: 1px solid rgba(255, 255, 255, 0.08);
        transform: translate(-50%, -0.15rem) rotate(45deg);
    }

    .tooltip::after {
        content: attr(data-tooltip);
        min-width: max-content;
        max-width: 13rem;
        padding: 0.35rem 0.5rem;
        border: 1px solid rgba(255, 255, 255, 0.08);
        border-radius: 6px;
        background-color: var(--primary-700);
        box-shadow: 0 6px 18px rgba(0, 0, 0, 0.35);
        color: var(--text-400);
        font-size: 0.68rem;
        font-weight: 600;
        line-height: 1.2;
        text-align: center;
        white-space: nowrap;
        transform: translate(-50%, -0.15rem);
    }

    .tooltip.align-start::before {
        left: 0.65rem;
        transform: translateY(-0.15rem) rotate(45deg);
    }

    .tooltip.align-start::after {
        left: 0;
        transform: translateY(-0.15rem);
    }

    .tooltip.align-end::before {
        left: auto;
        right: 0.65rem;
        transform: translateY(-0.15rem) rotate(45deg);
    }

    .tooltip.align-end::after {
        left: auto;
        right: 0;
        transform: translateY(-0.15rem);
    }

    .tooltip:hover::before,
    .tooltip:hover::after,
    .tooltip:focus-within::before,
    .tooltip:focus-within::after {
        opacity: 1;
    }

    .tooltip:hover::before,
    .tooltip:focus-within::before {
        transform: translate(-50%, 0) rotate(45deg);
    }

    .tooltip:hover::after,
    .tooltip:focus-within::after {
        transform: translate(-50%, 0);
    }

    .tooltip.align-start:hover::before,
    .tooltip.align-start:focus-within::before,
    .tooltip.align-end:hover::before,
    .tooltip.align-end:focus-within::before {
        transform: translateY(0) rotate(45deg);
    }

    .tooltip.align-start:hover::after,
    .tooltip.align-start:focus-within::after,
    .tooltip.align-end:hover::after,
    .tooltip.align-end:focus-within::after {
        transform: translateY(0);
    }
</style>
