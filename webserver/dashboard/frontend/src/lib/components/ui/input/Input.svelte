<script lang="ts">
    import type { Snippet } from "svelte";

    let {
        type = "text",
        label,
        pattern = ".*",
        value = $bindable(),
        placeholder,
        tooltip,
        maxCharacters = 30,
        // We want the behaviour the warning warns about.
        // eslint-disable-next-line
        // svelte-ignore state_referenced_locally
        boxWidthCh = maxCharacters,
        min,
        max,
        onSubmit,
        disabled = false,
        suffixElement,
    }: {
        type: "text" | "password" | "number";
        label: string;
        pattern?: string;
        value: string;
        placeholder?: string;
        tooltip?: string;
        maxCharacters?: number;
        boxWidthCh?: number;
        min?: number;
        max?: number;
        onSubmit?: (_event: Event) => void;
        disabled?: boolean;
        suffixElement?: Snippet;
    } = $props();

    let input: HTMLInputElement;

    let hasCustomError = false;

    export function setError(err: string) {
        input.setCustomValidity(err);
        input.reportValidity();
        hasCustomError = true;
    }

    $effect(() => {
        if (value && hasCustomError) {
            input.setCustomValidity("");
            hasCustomError = false;
        }
    });
</script>

<div class="input-label-container">
    <label class="label" for="{label}-{type}-input" title={tooltip}>{label}</label>
    <div class="input-container">
        <input
            bind:this={input}
            {type}
            class="input"
            id="{label}-{type}-input"
            {pattern}
            bind:value
            {placeholder}
            title={tooltip}
            onsubmit={onSubmit}
            maxlength={maxCharacters}
            {min}
            {max}
            style:--box-width-ch={boxWidthCh}
            {disabled}
        />
        {@render suffixElement?.()}
    </div>
</div>

<style>
    .input-container {
        display: flex;
        flex-direction: row;

        height: var(--input-height, 42px);
        padding: var(--input-padding, 0.5rem);

        border-width: var(--input-border-width, 1px);
        border-style: var(--input-border-style, solid);
        border-color: var(--input-border-color, var(--primary-450));
        border-radius: 10px;

        background-color: var(--primary-600);
    }

    .input-container:has(.input:focus) {
        border-color: var(--secondary-400);
    }

    .input-container:has(.input:invalid) {
        border-color: var(--error-border-color);
    }

    .input-label-container {
        display: flex;
        flex-direction: column;
        gap: 0.5rem;

        width: var(--input-width, 100%);

        margin: 1.2rem 0px;
    }

    .label {
        font-size: 1rem;
        color: var(--secondary-500);
        letter-spacing: 0.05em;
    }

    .input:disabled {
        filter: brightness(0.8);
    }

    .input {
        box-sizing: border-box;

        font-style: normal;
        font-family: "Chivo Mono Variable", monospace;
        font-size: 0.85rem;
        letter-spacing: var(--input-letter-spacing, 0.025em);
        color: var(--text-primary);
        background-color: var(--primary-600);

        width: 100%;
        height: 100%;

        transition-property: border-color;
        transition-timing-function: ease-in-out;
        transition-duration: 75ms;
    }
</style>
