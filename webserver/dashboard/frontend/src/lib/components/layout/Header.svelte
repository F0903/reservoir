<script lang="ts">
    import { getVersion } from "$lib/api/objects/version/version";
    import { onMount } from "svelte";

    let version_string = $state("");

    onMount(async () => {
        let version_obj = await getVersion();
        version_string = version_obj.version;
    });
</script>

<header>
    <h1>reservoir <span class="version-string">{version_string}</span></h1>
</header>

<style>
    .version-string {
        font-size: 0.7rem;
        color: var(--tertiary-700);
        user-select: none;
    }

    header {
        background-color: var(--primary-600);
        padding: 1rem;
    }

    h1 {
        color: var(--tertiary-400);
        user-select: none;
    }
</style>
