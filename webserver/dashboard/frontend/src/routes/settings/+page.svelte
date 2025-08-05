<script lang="ts">
    import TextInput from "$lib/components/ui/TextInput.svelte";
</script>

<h1 class="page-title">Settings</h1>
<TextInput
    label="Dashboard Update Interval"
    tooltip="The interval at which the dashboard updates its data from the API in milliseconds."
    value=""
    pattern="\d+"
    min={500}
/>

<style>
    .page-title {
        font-size: 1.75rem;
        font-weight: bold;
        color: var(--secondary-500);
        margin-bottom: 1rem;
    }
</style>
