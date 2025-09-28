<script lang="ts">
    let { onSubmit, children } = $props();

    function handleSubmit(event: Event) {
        event.preventDefault();
        onSubmit?.(event);
    }
</script>

<form onsubmit={handleSubmit}>
    {@render children()}
</form>
