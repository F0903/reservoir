<script lang="ts">
    let { title, children } = $props();
</script>

<div class="widget">
    <h2 class="title">{title}</h2>
    <div class="widget-content">
        {@render children()}
    </div>
</div>

<style>
    .title {
        margin-bottom: 1rem;

        font-size: 1.4rem;
        font-weight: 600;
        text-align: center;
    }

    .widget {
        width: fit-content;
        height: fit-content;

        padding: 1.5rem;
        padding-top: 1rem;
        border-radius: 15px;
        background-color: var(--primary-400);
    }
</style>
