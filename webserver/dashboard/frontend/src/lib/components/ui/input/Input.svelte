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
        {#if suffixElement}
            <div class="suffix">
                {@render suffixElement()}
            </div>
        {/if}
    </div>
</div>

<style>
    .input-container {
        display: flex;
        flex-direction: row;
        align-items: center;

        height: var(--input-height, 42px);
        padding: 0 0.75rem;

        border: 1px solid rgba(255, 255, 255, 0.08);
        border-radius: 8px;

        background-color: var(--primary-600);
        transition: all 0.2s ease;
    }

    .input-container:hover {
        border-color: rgba(255, 255, 255, 0.15);
    }

    .input-container:has(.input:focus) {
        border-color: var(--secondary-400);
        background-color: var(--primary-700);
        box-shadow: 0 0 0 2px rgba(var(--secondary-400-rgb), 0.1);
    }

    .input-container:has(.input:invalid) {
        border-color: var(--error-color);
    }

    .input-label-container {
        display: flex;
        flex-direction: column;
        gap: 0.4rem;
        width: var(--input-width, 100%);
        margin: 0.75rem 0;
    }

    .label {
        font-size: 0.75rem;
        font-weight: 700;
        text-transform: uppercase;
        letter-spacing: 0.08em;
        color: var(--secondary-300);
        opacity: 0.8;
        padding-left: 0.2rem;
    }

    .suffix {
        display: flex;
        align-items: center;
        margin-left: 0.5rem;
        opacity: 0.5;
    }

    .input:disabled {
        opacity: 0.5;
        cursor: not-allowed;
    }

    .input {
        box-sizing: border-box;
        font-style: normal;
        font-family: "Chivo Mono Variable", monospace;
        font-size: 0.9rem;
        color: var(--text-400);
        background: transparent;
        border: none;
        outline: none;

        width: 100%;
        height: 100%;
    }
</style>
