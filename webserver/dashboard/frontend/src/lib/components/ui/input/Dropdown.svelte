<script lang="ts">
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
</script>

<div class="input-container">
    <label class="label" for="{label}-dropdown" title={tooltip}>{label}</label>
    <select class="input" name="{label}-dropdown" id="{label}-dropdown" bind:value {disabled}>
        {#if !required}
            <option value="" selected={value === ""}>Select an option</option>
        {/if}
        {#each options as option (option)}
            <option value={option} selected={option === value}>{option}</option>
        {/each}
    </select>
</div>

<style>
    .input-container {
        display: flex;
        flex-direction: column;
        gap: 0.5rem;

        width: var(--dropdown-width, 100%);

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
        --dropdown-padding: 0.5rem;
        --dropdown-border-width: 1px;
        --dropdown-letter-spacing: 0.025em;

        padding: var(--dropdown-padding);
        border: 1px solid var(--primary-450);
        border-radius: 10px;
        box-sizing: border-box;

        font-style: normal;
        font-family: "Chivo Mono Variable", monospace;
        font-size: 0.85rem;
        letter-spacing: var(--dropdown-letter-spacing);
        color: var(--text-primary);
        background-color: var(--primary-600);

        width: default;

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
