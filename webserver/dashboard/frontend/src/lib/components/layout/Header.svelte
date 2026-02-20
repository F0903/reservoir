<script lang="ts">
    import { version } from "$lib/api/objects/version/version";
    import { onMount } from "svelte";
    import { getAuthProvider } from "$lib/context";
    import { LogOut, User, Menu } from "@lucide/svelte";

    let {
        onToggleMenu,
    }: {
        onToggleMenu?: () => void;
    } = $props();

    let version_string = $state("");
    const session = getAuthProvider();

    onMount(async () => {
        let version_obj = await version();
        version_string = version_obj.version;
    });
</script>

<header>
    <div class="title-section">
        <button class="menu-toggle" onclick={onToggleMenu} aria-label="Toggle menu">
            <Menu size={24} />
        </button>
        <h1>reservoir <span class="version-string">{version_string}</span></h1>
    </div>

    {#if session.user}
        <div class="user-section">
            <div class="user-info">
                <User size={18} />
                <span class="username">{session.user.username}</span>
            </div>
            <button class="logout-btn" onclick={session.logout} title="Logout">
                <LogOut size={18} />
                <span class="logout-text">Logout</span>
            </button>
        </div>
    {/if}
</header>

<style>
    .version-string {
        font-size: 0.7rem;
        color: var(--tertiary-700);
        user-select: none;
    }

    header {
        background-color: var(--primary-600);
        padding: 0.75rem 1.5rem;
        display: flex;
        justify-content: space-between;
        align-items: center;
    }

    .title-section {
        display: flex;
        align-items: center;
        gap: 1rem;
    }

    .menu-toggle {
        display: none;
        color: var(--text-400);
        padding: 0.5rem;
        margin-left: -0.5rem;
        border-radius: 8px;
        transition: background-color 0.2s;
    }

    .menu-toggle:hover {
        background-color: rgba(255, 255, 255, 0.1);
    }

    h1 {
        color: var(--tertiary-400);
        user-select: none;
        font-size: 1.5rem;
        font-weight: 700;
    }

    .user-section {
        display: flex;
        align-items: center;
        gap: 1.5rem;
    }

    .user-info {
        display: flex;
        align-items: center;
        gap: 0.5rem;
        color: var(--text-400);
        font-size: 0.9rem;
    }

    .username {
        font-weight: 600;
    }

    .logout-btn {
        display: flex;
        align-items: center;
        gap: 0.4rem;
        color: var(--secondary-300);
        font-size: 0.9rem;
        font-weight: 600;
        background: rgba(255, 255, 255, 0.05);
        padding: 0.4rem 0.8rem;
        border-radius: 8px;
        transition: all 0.2s ease;
    }

    .logout-btn:hover {
        background: rgba(255, 255, 255, 0.1);
        color: var(--secondary-400);
    }

    @media (max-width: 768px) {
        header {
            padding: 0.75rem 1rem;
        }

        .menu-toggle {
            display: flex;
        }

        .user-info {
            display: none;
        }

        .logout-text {
            display: none;
        }

        .logout-btn {
            padding: 0.5rem;
        }
    }
</style>
