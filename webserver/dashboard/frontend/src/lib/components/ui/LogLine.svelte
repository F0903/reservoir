<script lang="ts">
    let { line }: { line: string } = $props();

    function parseLine(text: string) {
        const levelMatch = text.match(/level=([^\s]+)/);
        const level = levelMatch ? levelMatch[1].toUpperCase() : "INFO";

        // This regex matches key=value pairs, handling quoted values
        // It's a bit simplified but works for standard slog output
        const parts: { text: string; type: "level" | "key" | "value" | "text" }[] = [];

        // Regex to match:
        // 1. level=VALUE
        // 2. KEY=VALUE (where VALUE can be "..." or a string without spaces)
        // 3. Any other text
        const regex = /(level=[^\s]+)|(\w+)=("[^"]*"|[^\s]+)|([^\s]+)/g;
        let lastIndex = 0;
        let match;

        while ((match = regex.exec(text)) !== null) {
            // Add whitespace between matches if any
            if (match.index > lastIndex) {
                parts.push({ text: text.slice(lastIndex, match.index), type: "text" });
            }

            if (match[1]) {
                parts.push({ text: match[1], type: "level" });
            } else if (match[2] && match[3]) {
                parts.push({ text: match[2], type: "key" });
                parts.push({ text: "=", type: "text" });
                parts.push({ text: match[3], type: "value" });
            } else if (match[4]) {
                parts.push({ text: match[4], type: "text" });
            }
            lastIndex = regex.lastIndex;
        }

        // Add any remaining text
        if (lastIndex < text.length) {
            parts.push({ text: text.slice(lastIndex), type: "text" });
        }

        return { level, parts };
    }

    const parsed = $derived(parseLine(line));
</script>

<div class="log-line" data-level={parsed.level}>
    <!-- eslint-disable-next-line -->
    {#each parsed.parts as part}
        <span class="part-{part.type}">{part.text}</span>
    {/each}
</div>

<style>
    .log-line {
        font-family: "Chivo Mono Variable", monospace;
        font-size: 0.85rem;
        white-space: pre-wrap;
        word-break: break-all;
        padding: 4px 8px;
        border-bottom: 1px solid var(--primary-450);
        color: var(--text-400);
        line-height: 1.4;
    }

    .log-line:hover {
        background-color: var(--primary-450);
    }

    .log-line:last-child {
        border-bottom: none;
    }

    .part-level {
        font-weight: bold;
        padding: 0 4px;
        border-radius: 4px;
        margin-right: 4px;
    }

    .part-key {
        color: var(--tertiary-600);
        font-weight: 500;
    }

    .part-value {
        color: var(--secondary-300);
    }

    .part-text {
        color: var(--text-900);
    }

    [data-level="INFO"] .part-level {
        background-color: hsla(120, 50%, 70%, 0.1);
        color: var(--log-info-color);
    }
    [data-level="WARN"] .part-level {
        background-color: hsla(45, 100%, 70%, 0.1);
        color: var(--log-warn-color);
    }
    [data-level="ERROR"] .part-level {
        background-color: hsla(0, 100%, 70%, 0.1);
        color: var(--log-error-color);
    }
    [data-level="DEBUG"] .part-level {
        background-color: hsla(210, 20%, 60%, 0.1);
        color: var(--log-debug-color);
    }
</style>
