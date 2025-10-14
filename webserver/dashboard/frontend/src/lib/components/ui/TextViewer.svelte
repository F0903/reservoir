<script lang="ts">
    let {
        initialContent,
        scrollMargin = 20,
    }: {
        initialContent?: string;
        scrollMargin?: number; // The margin of which the viewer needs to be manually scrolled up to avoid auto-scroll.
    } = $props();

    let viewer: HTMLDivElement;
    let textElem: HTMLPreElement;

    export function appendText(newText: string) {
        textElem.textContent += newText;

        if (viewer.scrollTop + viewer.clientHeight >= viewer.scrollHeight - scrollMargin) {
            viewer.scrollTop = viewer.scrollHeight;
        }
    }
</script>

<div class="viewer" bind:this={viewer}>
    <pre bind:this={textElem}>{initialContent}</pre>
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
