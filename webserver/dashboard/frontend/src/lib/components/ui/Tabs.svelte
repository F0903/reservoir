<script lang="ts" generics="T extends string">
    import type { IconProps } from "@lucide/svelte";
    import type { Component, Snippet } from "svelte";
    import { fade } from "svelte/transition";

    let {
        tabs,
        activeTab = $bindable(),
        ...rest
    }: {
        tabs: ReadonlyArray<{ id: T; label: string; icon: Component<IconProps> }>;
        activeTab: T;
        [key: string]: unknown;
    } = $props();
</script>

<div class="tabs-container">
    <nav class="tab-bar">
        {#each tabs as tab (tab.id)}
            <button
                class="tab-btn"
                class:active={activeTab === tab.id}
                onclick={() => (activeTab = tab.id)}
            >
                <tab.icon size={18} />
                <span>{tab.label}</span>
            </button>
        {/each}
    </nav>

    <div class="tab-content" in:fade={{ duration: 200 }} data-key={activeTab}>
        {#if rest[activeTab]}
            {@render (rest[activeTab] as Snippet)()}
        {/if}
    </div>
</div>

<style>
    .tabs-container {
        display: flex;
        flex-direction: column;
        background-color: var(--primary-450);
        border-radius: 20px;
        border: 1px solid rgba(255, 255, 255, 0.05);
        overflow: hidden;
        box-shadow: 0 10px 30px rgba(0, 0, 0, 0.2);
        width: 100%;
    }

    .tab-bar {
        display: flex;
        background-color: rgba(0, 0, 0, 0.2);
        padding: 0.5rem;
        gap: 0.5rem;
        flex-wrap: wrap;
    }

    .tab-btn {
        display: flex;
        align-items: center;
        gap: 0.6rem;
        padding: 0.75rem 1.25rem;
        border-radius: 12px;
        color: var(--text-400);
        opacity: 0.6;
        font-weight: 600;
        transition: all 0.2s ease;
        flex: 1 1 auto;
        justify-content: center;
    }

    .tab-btn:hover {
        opacity: 1;
        background-color: rgba(255, 255, 255, 0.05);
    }

    .tab-btn.active {
        opacity: 1;
        background-color: var(--primary-300);
        color: var(--secondary-300);
        box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
    }

    .tab-content {
        padding: 1.5rem 2.5rem;
        width: 100%;
        display: flex;
        flex-direction: column;
    }

    @media (max-width: 768px) {
        .tab-content {
            padding: 1rem;
        }

        .tab-btn {
            padding: 0.6rem 0.8rem;
            font-size: 0.9rem;
        }

        .tab-btn span {
            display: none;
        }
    }
</style>
