<script lang="ts">
    import { onMount, type Snippet } from "svelte";

    let {
        text,
        align = "center",
        children,
    }: {
        text: string;
        align?: "start" | "center" | "end";
        children: Snippet;
    } = $props();

    let trigger: HTMLSpanElement;
    let open = $state(false);

    onMount(() => {
        trigger.addEventListener("focusin", showTooltip);
        trigger.addEventListener("focusout", handleFocusOut);
        trigger.addEventListener("keydown", handleKeydown);
        trigger.addEventListener("pointerenter", showTooltip);
        trigger.addEventListener("pointerleave", hideTooltip);

        return () => {
            trigger.removeEventListener("focusin", showTooltip);
            trigger.removeEventListener("focusout", handleFocusOut);
            trigger.removeEventListener("keydown", handleKeydown);
            trigger.removeEventListener("pointerenter", showTooltip);
            trigger.removeEventListener("pointerleave", hideTooltip);
        };
    });

    $effect(() => {
        if (!open) return;

        updatePosition();
        window.addEventListener("resize", updatePosition);
        window.addEventListener("scroll", updatePosition, true);

        return () => {
            window.removeEventListener("resize", updatePosition);
            window.removeEventListener("scroll", updatePosition, true);
        };
    });

    function showTooltip() {
        open = true;
        updatePosition();
    }

    function hideTooltip() {
        open = false;
    }

    function handleFocusOut(event: FocusEvent) {
        const nextTarget = event.relatedTarget;
        if (nextTarget instanceof Node && trigger.contains(nextTarget)) {
            return;
        }

        hideTooltip();
    }

    function handleKeydown(event: KeyboardEvent) {
        if (event.key === "Escape") {
            hideTooltip();
        }
    }

    function updatePosition() {
        if (!trigger) return;

        const rect = trigger.getBoundingClientRect();
        const anchorOffset = 10;
        const top = rect.bottom + 7;
        const labelLeft =
            align === "start"
                ? rect.left
                : align === "end"
                  ? rect.right
                  : rect.left + rect.width / 2;
        const arrowLeft =
            align === "start"
                ? rect.left + anchorOffset
                : align === "end"
                  ? rect.right - anchorOffset
                  : rect.left + rect.width / 2;

        trigger.style.setProperty("--tooltip-top", `${top}px`);
        trigger.style.setProperty("--tooltip-left", `${labelLeft}px`);
        trigger.style.setProperty("--tooltip-arrow-left", `${arrowLeft}px`);
    }
</script>

<span
    bind:this={trigger}
    class="tooltip"
    class:align-start={align === "start"}
    class:align-end={align === "end"}
>
    {@render children()}
    {#if open}
        <span class="tooltip-arrow" aria-hidden="true"></span>
        <span class="tooltip-label" aria-hidden="true">{text}</span>
    {/if}
</span>

<style>
    .tooltip {
        --tooltip-label-x: -50%;
        display: inline-grid;
        place-items: center;
    }

    .tooltip.align-start {
        --tooltip-label-x: 0;
    }

    .tooltip.align-end {
        --tooltip-label-x: -100%;
    }

    .tooltip-arrow,
    .tooltip-label {
        position: fixed;
        top: var(--tooltip-top);
        z-index: 100;
        pointer-events: none;
    }

    .tooltip-arrow {
        left: var(--tooltip-arrow-left);
        width: 0.45rem;
        height: 0.45rem;
        background-color: var(--primary-700);
        border-left: 1px solid rgba(255, 255, 255, 0.08);
        border-top: 1px solid rgba(255, 255, 255, 0.08);
        animation: tooltip-arrow-in 120ms ease both;
    }

    .tooltip-label {
        left: var(--tooltip-left);
        width: max-content;
        max-width: min(13rem, calc(100vw - 1rem));
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
        animation: tooltip-label-in 120ms ease both;
    }

    @keyframes tooltip-arrow-in {
        from {
            opacity: 0;
            transform: translate(-50%, -0.15rem) rotate(45deg);
        }

        to {
            opacity: 1;
            transform: translate(-50%, 0) rotate(45deg);
        }
    }

    @keyframes tooltip-label-in {
        from {
            opacity: 0;
            transform: translate(var(--tooltip-label-x), -0.15rem);
        }

        to {
            opacity: 1;
            transform: translate(var(--tooltip-label-x), 0);
        }
    }
</style>
