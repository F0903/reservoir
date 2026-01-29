<script lang="ts">
    import { tick } from "svelte";
    import LogLine from "./LogLine.svelte";

    let {
        initialContent = "",
        scrollMargin = 20,
        autoScroll = true,
        isLogViewer = false,
    }: {
        initialContent?: string;
        scrollMargin?: number;
        autoScroll?: boolean;
        isLogViewer?: boolean;
    } = $props();

    let viewer: HTMLDivElement;

    let lines = $state<string[]>([]);

    $effect(() => {
        if (initialContent) {
            const newLines = initialContent.split("\n").filter((l) => l.length > 0);
            lines = [...newLines];
        }
    });

    $effect(() => {
        if (autoScroll && lines.length > 0) {
            scrollViewer();
        }
    });

    async function scrollViewer() {
        if (!viewer) return;
        await tick();
        viewer.scrollTop = viewer.scrollHeight;
    }

    export async function appendText(newText: string) {
        if (!newText) return;

        const newLines = newText.split("\n").filter((l) => l.length > 0);
        lines.push(...newLines);

        if (lines.length > 2000) {
            lines = lines.slice(lines.length - 2000);
        }
    }

    export function clear() {
        lines = [];
    }
</script>

<div class="viewer" bind:this={viewer}>
    {#if isLogViewer}
        <div class="log-container">
            {#each lines as line}
                <LogLine {line} />
            {/each}
        </div>
    {:else}
        <pre>{lines.join("\n")}</pre>
    {/if}
</div>

<style>
    .viewer {
        border: 1px solid var(--secondary-600);
        border-radius: 12px;
        padding: 13px 15px;
        background-color: var(--primary-600);
        overflow-y: auto;
        max-height: var(--viewer-max-height, 100%);
    }

    .log-container {
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
