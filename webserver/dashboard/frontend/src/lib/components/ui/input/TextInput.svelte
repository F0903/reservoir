<script lang="ts">
    let {
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
        censor = false,
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
        onSubmit?: (_event: Event) => void;
        disabled?: boolean;
        censor?: boolean;
    } = $props();

    let input: HTMLInputElement | undefined = $state();

    let hasCustomError = false;

    export function setError(err: string) {
        input!.setCustomValidity(err);
        input!.reportValidity();
        hasCustomError = true;
    }

    $effect(() => {
        if (value && hasCustomError) {
            input!.setCustomValidity("");
            hasCustomError = false;
        }
    });
</script>

<div class="input-container">
    <label class="label" for="{label}-textinput" title={tooltip}>{label}</label>
    <input
        bind:this={input}
        type={censor ? "password" : "text"}
        class="input"
        id="{label}-textinput"
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
</div>

<style>
    .input-container {
        display: flex;
        flex-direction: column;
        gap: 0.5rem;

        width: var(--textinput-width, 100%);

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

        width: 100%;

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
