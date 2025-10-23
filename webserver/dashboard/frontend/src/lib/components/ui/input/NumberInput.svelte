<script lang="ts">
    import TextInput from "./TextInput.svelte";

    let {
        label,
        value: valProp = $bindable(),
        placeholder,
        tooltip,
        maxCharacters = 30,
        boxWidthCh = maxCharacters,
        min,
        max,
        onSubmit,
        disabled = false,
        censor = false,
    }: {
        label: string;
        value: number;
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

    const intPattern = "^\\d+$";
</script>

<TextInput
    {label}
    bind:value={() => valProp.toString(), (val: unknown) => (valProp = Number(val))}
    pattern={intPattern}
    {placeholder}
    {tooltip}
    {maxCharacters}
    {boxWidthCh}
    {min}
    {max}
    {onSubmit}
    {disabled}
    {censor}
/>
