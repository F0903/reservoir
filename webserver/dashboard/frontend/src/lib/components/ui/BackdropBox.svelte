<script lang="ts">
    let { children, class: className = "" } = $props();
</script>

<div class="box-container {className}">
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
        background-color: var(--box-backdrop-color, var(--primary-600));
    }

    .box {
        width: var(--box-width, 100%);
        height: var(--box-height, 100%);
        border-radius: var(--box-border-radius);
        background-color: var(--box-background-color, var(--primary-500));
        box-shadow: var(--box-shadow, var(--default-shadow));
    }
</style>
