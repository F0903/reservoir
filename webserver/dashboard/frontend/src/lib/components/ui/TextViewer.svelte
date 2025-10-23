<script lang="ts">
    import { tick } from "svelte";

    let {
        initialContent,
        scrollMargin = 20,
        autoScroll = true,
    }: {
        initialContent?: string;
        scrollMargin?: number; // The margin of which the viewer needs to be manually scrolled up to avoid auto-scroll.
        autoScroll?: boolean; // Whether the viewer should automatically scroll to the bottom when new text is appended.
    } = $props();

    let viewer: HTMLDivElement;

    let text = $state(initialContent);

    $effect.pre(() => {
        if (autoScroll && text) {
            scrollViewer();
        }
    });

    async function scrollViewer() {
        if (viewer.scrollTop + viewer.clientHeight >= viewer.scrollHeight - scrollMargin) {
            await tick(); // Wait for the DOM to update before scrolling
            viewer.scrollTop = viewer.scrollHeight;
        }
    }

    export async function appendText(newText: string) {
        text += newText;
    }
</script>

<div class="viewer" bind:this={viewer}>
    <pre>{text}</pre>
</div>

<style>
    .viewer {
        border-color: var(--secondary-600);
        border-radius: 12px;
        padding: 13px 15px;
        background-color: var(--primary-600);
        overflow-y: auto;
        max-height: var(--viewer-max-height, 100%);
    }
</style>
