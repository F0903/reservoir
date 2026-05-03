<script lang="ts">
    import { goto } from "$app/navigation";
    import { resolve } from "$app/paths";
    import { version } from "$lib/api/objects/version/version";
    import { userIsAdmin } from "$lib/auth/permissions";
    import { onMount } from "svelte";
    import { getAuthProvider } from "$lib/context";
    import { LogOut, Menu, ShieldCheck, User } from "@lucide/svelte";

    let {
        onToggleMenu,
    }: {
        onToggleMenu?: () => void;
    } = $props();

    let version_string = $state("");
    const session = getAuthProvider();
    const isAdmin = $derived(userIsAdmin(session.user));

    onMount(async () => {
        let version_obj = await version();
        version_string = version_obj.version;
    });

    async function openUserPage() {
        await goto(resolve("/dashboard/user"));
    }
</script>

<header>
    <div class="title-section">
        {#if session.user && onToggleMenu}
            <button class="menu-toggle" onclick={onToggleMenu} aria-label="Toggle menu">
                <Menu size={24} />
            </button>
        {/if}
        <h1>reservoir <span class="version-string">{version_string}</span></h1>
    </div>

    {#if session.user}
        <div class="user-section">
            <button class="user-info" onclick={openUserPage} aria-label="Open user profile">
                <User size={18} />
                <span class="username">{session.user.username}</span>
                {#if isAdmin}
                    <span class="admin-badge" aria-label="Administrator">
                        <ShieldCheck size={13} />
                        <span class="admin-badge-text">Admin</span>
                    </span>
                {/if}
            </button>
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
        padding: 0.35rem 0.45rem;
        border: 0;
        border-radius: 8px;
        background: transparent;
        color: var(--text-400);
        cursor: pointer;
        font-size: 0.9rem;
        transition:
            background-color 120ms ease,
            color 120ms ease;
    }

    .user-info:hover {
        background-color: rgba(255, 255, 255, 0.05);
        color: var(--secondary-300);
    }

    .username {
        font-weight: 600;
    }

    .admin-badge {
        display: inline-flex;
        align-items: center;
        gap: 0.25rem;
        min-height: 1.35rem;
        padding: 0.15rem 0.45rem;
        border: 1px solid color-mix(in srgb, var(--secondary-300) 26%, transparent);
        border-radius: 999px;
        background-color: color-mix(in srgb, var(--secondary-800) 24%, transparent);
        color: var(--secondary-300);
        font-size: 0.64rem;
        font-weight: 800;
        line-height: 1;
        text-transform: uppercase;
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

        .username {
            display: none;
        }

        .user-info {
            gap: 0.35rem;
        }

        .admin-badge {
            padding: 0.25rem;
        }

        .admin-badge-text {
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
