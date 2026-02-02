<script lang="ts">
    import { onMount, tick } from "svelte";
    import LogLine from "./LogLine.svelte";
    import { isNumber } from "$lib/utils/generic-type-predicates";

    type SyntaxHighlightingType = "none" | "slog";

    let {
        initialContent = "",
        autoScrollDisableMargin = 20,
        autoScroll = $bindable(true),
        syntaxHighlighting = "none",
    }: {
        initialContent?: string;
        autoScrollDisableMargin?: number | false;
        autoScroll?: boolean;
        syntaxHighlighting?: SyntaxHighlightingType;
    } = $props();

    let viewer: HTMLDivElement;

    type LineEntry = { id: number; text: string };
    let lines = $state<LineEntry[]>([]);
    let nextId = 0;

    onMount(() => {
        if (initialContent) {
            appendText(initialContent);
        }
    });

    function handleScroll() {
        if (!viewer) return;

        const isAtBottom =
            isNumber(autoScrollDisableMargin) &&
            viewer.scrollTop + viewer.clientHeight >= viewer.scrollHeight - autoScrollDisableMargin;

        if (!isAtBottom && autoScroll) {
            autoScroll = false;
            console.log("Auto scroll disabled");
        }
    }

    async function scrollToBottom() {
        await tick();

        viewer.scrollTop = viewer.scrollHeight;
    }

    export async function appendText(text: string) {
        if (!text) return;
        const rawLines = text.split("\n").filter((l) => l.length > 0);
        const newEntries = rawLines.map((text) => ({ id: nextId++, text }));

        lines.push(...newEntries);

        if (lines.length > 2000) {
            lines = lines.slice(lines.length - 2000);
        }

        if (autoScroll) {
            await scrollToBottom();
        }
    }

    export function clear() {
        lines = [];
    }
</script>

<div class="viewer" bind:this={viewer} onscroll={handleScroll}>
    {#if syntaxHighlighting === "slog"}
        <div class="slog-container">
            {#each lines as line (line.id)}
                <LogLine line={line.text} />
            {/each}
        </div>
    {:else}
        <pre>{lines.map((l) => l.text).join("\n")}</pre>
    {/if}
</div>

<style>
    .viewer {
        border: 1px solid var(--secondary-800);
        border-radius: 8px;
        padding: 13px 15px;
        background-color: var(--primary-600);
        overflow-y: auto;
        max-height: var(--viewer-max-height, 100%);
    }

    .slog-container {
        display: flex;
        flex-direction: column;
    }

    pre {
        margin: 0;
        font-family: "Chivo Mono Variable", monospace;
        font-size: 0.85rem;
        white-space: pre-wrap;
        word-break: break-all;
    }
</style>
