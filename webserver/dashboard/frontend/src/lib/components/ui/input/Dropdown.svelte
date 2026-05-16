<script lang="ts">
    import InputLabel from "./InputLabel.svelte";

    let {
        label,
        options,
        value = $bindable(),
        required = true,
        tooltip,
        disabled = false,
    }: {
        label: string;
        options: string[];
        value: string;
        required?: boolean;
        tooltip?: string;
        disabled?: boolean;
    } = $props();

    const inputId = $derived(`${label}-dropdown`);
</script>

<div class="input-label-container">
    <InputLabel {label} {tooltip} {inputId} />
    <div class="input-container">
        <select class="input" name={inputId} id={inputId} bind:value {disabled}>
            {#if !required}
                <option value="" selected={value === ""}>Select an option</option>
            {/if}
            {#each options as option (option)}
                <option value={option} selected={option === value}>{option}</option>
            {/each}
        </select>
    </div>
</div>

<style>
    .input-label-container {
        display: flex;
        flex-direction: column;
        gap: 0.4rem;
        width: var(--dropdown-width, 100%);
        margin: 0.75rem 0;
    }

    .input-container {
        display: flex;
        flex-direction: row;
        align-items: center;

        height: var(--dropdown-height, 42px);
        padding: 0 0.5rem;

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
        cursor: pointer;
    }
</style>
