<script lang="ts">
    import Input from "./Input.svelte";

    let {
        label,
        value: valProp = $bindable(),
        placeholder,
        tooltip,
        min = 0,
        max = 100,
        onSubmit,
        disabled = false,
    }: {
        label: string;
        value: number;
        placeholder?: string;
        tooltip?: string;
        min?: number;
        max?: number;
        onSubmit?: (_event: Event) => void;
        disabled?: boolean;
    } = $props();
</script>

<Input
    type="number"
    {label}
    bind:value={() => valProp?.toString(), (val: string) => (valProp = Number(val))}
    {placeholder}
    {tooltip}
    {min}
    {max}
    {onSubmit}
    {disabled}
>
    {#snippet suffixElement()}
        <div class="percent">%</div>
    {/snippet}
</Input>

<style>
    .percent {
        margin: auto;
        padding: 0px 5px;
        background-color: var(--primary-600);
    }
</style>
