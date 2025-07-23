<script lang="ts">
    let { children } = $props();
</script>

<div class="box-container">
    <div class="backdrop"></div>
    <div class="box">
        {@render children()}
    </div>
</div>

<style>
    .box-container {
        position: relative;
        width: var(--box-width, 100%);
        height: var(--box-height, 100%);
    }

    .backdrop {
        z-index: -1;
        position: absolute;
        top: 0;
        left: 0;
        width: var(--box-width, 100%);
        height: var(--box-height, 100%);
        background-color: var(--box-backdrop-color, var(--background-secondary));
    }

    .box {
        width: var(--box-width, 100%);
        height: var(--box-height, 100%);
        border-radius: var(--box-border-radius);
        background-color: var(--box-background-color, var(--background-primary));
        box-shadow: var(--box-shadow, none);
    }
</style>
