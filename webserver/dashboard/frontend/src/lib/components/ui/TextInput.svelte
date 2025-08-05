<script lang="ts">
    import { log } from "$lib/utils/logger";
    import type { HTMLInputTypeAttribute } from "svelte/elements";

    let {
        label,
        pattern = ".*",
        value = $bindable(),
        placeholder,
        tooltip,
        maxCharacters = 30,
        boxWidthCh = maxCharacters,
        min,
        max,
        onSubmit,
    }: {
        label: string;
        pattern?: string;
        value: string;
        placeholder?: string;
        tooltip?: string;
        maxCharacters?: number;
        boxWidthCh?: number;
        min?: number;
        max?: number;
        onSubmit?: (event: Event) => void;
    } = $props();
</script>

<div class="input-container">
    <label class="label" for="{label}-input" title={tooltip}>{label}</label>
    <input
        type="text"
        {pattern}
        class="input"
        id="{label}-input"
        bind:value
        {placeholder}
        title={tooltip}
        onsubmit={onSubmit}
        maxlength={maxCharacters}
        {min}
        {max}
        style:--box-width-ch={boxWidthCh}
    />
</div>

<style>
    .input-container {
        display: flex;
        flex-direction: column;
        gap: 0.5rem;

        width: fit-content;

        margin: 1.2rem 0px;
    }

    .label {
        font-size: 1rem;
        color: var(--secondary-500);
        letter-spacing: 0.05em;
    }

    .input {
        --text-input-padding: 0.5rem;
        --text-input-border-width: 1px;
        --text-input-letter-spacing: 0.025em;

        padding: var(--text-input-padding);
        border: 1px solid var(--primary-450);
        border-radius: 10px;
        box-sizing: border-box;

        font-style: normal;
        font-family: "Chivo Mono Variable", monospace;
        font-size: 0.85rem;
        letter-spacing: var(--text-input-letter-spacing);
        color: var(--text-primary);
        background-color: var(--primary-600);

        width: calc(
            var(--box-width-ch) * (1ch + calc(var(--text-input-letter-spacing))) +
                calc(var(--text-input-padding) * 2) + calc(var(--text-input-border-width) * 2)
        );

        transition-property: border-color;
        transition-timing-function: ease-in-out;
        transition-duration: 75ms;
    }

    .input:focus {
        border-color: var(--secondary-400);
    }

    .input:invalid {
        border-color: var(--error-border-color);
    }
</style>
