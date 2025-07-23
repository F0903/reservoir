<script lang="ts">
    let { onClick, disabled = false, children } = $props();
</script>

<button class="btn" onclick={onClick} {disabled}>
    {@render children()}
</button>

<style>
    .btn:enabled:hover {
        filter: brightness(0.9);
    }

    .btn:enabled:active {
        filter: brightness(0.8);
        scale: 0.95;
    }

    .btn {
        padding: 0.5rem 0.75rem;
        border: none;
        border-radius: 15px;

        cursor: pointer;
        background-color: var(--btn-background-color, var(--primary-color));

        color: var(--btn-text-color, var(--text-primary));
        font-size: 1rem;
        text-align: center;

        transition-property: filter, scale;
        transition-duration: var(--btn-transition-duration, 100ms);
        transition-timing-function: var(--btn-transition-timing, cubic-bezier(0.19, 1, 0.22, 1));
    }
</style>
