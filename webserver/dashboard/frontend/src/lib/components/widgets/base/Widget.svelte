<script lang="ts">
    import Card from "$lib/components/ui/Card.svelte";

    let { title, children } = $props();
</script>

<Card --card-width="fit-content" --card-height="fit-content">
    <h2 class="title">{title}</h2>
    <div class="widget-content">
        {@render children()}
    </div>
</Card>

<style>
    .title {
        margin-top: -5px;
        margin-bottom: 1rem;

        font-size: 1.4rem;
        font-weight: 600;
        text-align: center;
    }
</style>
