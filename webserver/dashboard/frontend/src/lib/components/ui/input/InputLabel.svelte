<script lang="ts">
    import Tooltip from "$lib/components/ui/Tooltip.svelte";

    let {
        label,
        inputId,
        tooltip,
    }: {
        label: string;
        inputId: string;
        tooltip?: string;
    } = $props();
</script>

{#if tooltip}
    <Tooltip text={tooltip} align="start">
        <label class="input-label" for={inputId}>{label}</label>
    </Tooltip>
{:else}
    <label class="input-label" for={inputId}>{label}</label>
{/if}

<style>
    .input-label {
        font-size: 0.75rem;
        font-weight: 700;
        text-transform: uppercase;
        letter-spacing: 0.08em;
        color: var(--secondary-300);
        opacity: 0.8;
        padding-left: 0.2rem;
    }
</style>
