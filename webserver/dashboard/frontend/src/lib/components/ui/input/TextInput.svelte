<script lang="ts">
    import type { Snippet } from "svelte";
    import Input from "./Input.svelte";

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
        suffixElement,
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
        suffixElement?: Snippet;
    } = $props();

    let input: Input;

    export function setError(err: string) {
        input.setError(err);
    }
</script>

<Input
    bind:this={input}
    type={censor ? "password" : "text"}
    {label}
    {pattern}
    bind:value
    {placeholder}
    {tooltip}
    {maxCharacters}
    {boxWidthCh}
    {min}
    {max}
    {onSubmit}
    {disabled}
    {suffixElement}
/>
