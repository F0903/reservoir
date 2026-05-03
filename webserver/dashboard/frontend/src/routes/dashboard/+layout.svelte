<script lang="ts">
    import { browser } from "$app/environment";
    import { goto } from "$app/navigation";
    import { resolve } from "$app/paths";
    import { page } from "$app/state";
    import Header from "$lib/components/layout/Header.svelte";
    import SideNav from "$lib/components/layout/SideNav.svelte";
    import SideNavButton from "$lib/components/layout/SideNavButton.svelte";
    import BackdropBox from "$lib/components/ui/BackdropBox.svelte";
    import { userIsAdmin } from "$lib/auth/permissions";
    import { getAuthProvider } from "$lib/context";
    import { LayoutDashboard, Logs, Settings, UsersRound } from "@lucide/svelte";

    let { children } = $props();

    const auth = getAuthProvider();
    const isAdmin = $derived(userIsAdmin(auth.user));
    let isMenuOpen = $state(false);

    $effect(() => {
        if (!browser || auth.loading || !auth.user || isAdmin) {
            return;
        }

        const dashboardPath = resolve("/dashboard");
        if (isAdminRoute(page.url.pathname)) {
            void goto(dashboardPath, { replaceState: true });
        }
    });

    function toggleMenu() {
        isMenuOpen = !isMenuOpen;
    }

    function closeMenu() {
        isMenuOpen = false;
    }

    function isAdminRoute(pathname: string) {
        return [resolve("/dashboard/settings"), resolve("/dashboard/users")].some((path) =>
            isPathWithin(pathname, path),
        );
    }

    function isPathWithin(pathname: string, root: string) {
        return pathname === root || pathname.startsWith(`${root}/`);
    }
</script>

<div class="layout-grid" class:no-sidenav={!auth.user}>
    <div class="header-area">
        <Header onToggleMenu={toggleMenu} />
    </div>
    {#if auth.user}
        <div class="sidenav-area" class:open={isMenuOpen}>
            <SideNav>
                <SideNavButton url="/dashboard" onClick={closeMenu}>
                    <LayoutDashboard />Dashboard
                </SideNavButton>
                {#if isAdmin}
                    <SideNavButton url="/dashboard/settings" onClick={closeMenu}>
                        <Settings />Settings
                    </SideNavButton>
                    <SideNavButton url="/dashboard/users" onClick={closeMenu}>
                        <UsersRound />Users
                    </SideNavButton>
                {/if}
                <SideNavButton url="/dashboard/log" onClick={closeMenu}>
                    <Logs />Log
                </SideNavButton>
            </SideNav>
        </div>
    {/if}

    {#if auth.user && isMenuOpen}
        <!-- svelte-ignore a11y_click_events_have_key_events -->
        <!-- svelte-ignore a11y_no_static_element_interactions -->
        <div class="menu-backdrop" onclick={closeMenu}></div>
    {/if}

    <div class="main-area">
        <BackdropBox --box-border-radius="20px 0px 0px 0px" class="main-backdrop">
            <div class="page-container">
                {@render children()}
            </div>
        </BackdropBox>
    </div>
</div>

<style>
    .layout-grid {
        display: grid;
        grid-template-columns: auto 1fr;
        grid-template-rows: auto 1fr;
        grid-template-areas:
            "header header"
            "sidenav main";
        gap: 0;
        height: 100%;
    }

    .header-area {
        grid-area: header;
        min-height: 0;
        min-width: 0;
        width: 100%;
        z-index: 50;
    }

    .layout-grid.no-sidenav {
        grid-template-columns: 1fr;
        grid-template-areas:
            "header"
            "main";
    }

    .sidenav-area {
        grid-area: sidenav;
        min-height: 0;
        min-width: 0;
        height: 100%;
        transition: transform 0.3s ease;
        z-index: 40;
    }

    .main-area {
        grid-area: main;
        min-height: 0;
        min-width: 0;
        height: 100%;
    }

    .page-container {
        padding: 2rem;
        overflow-y: auto;
        height: 100%;
    }

    @media (max-width: 768px) {
        .layout-grid {
            grid-template-columns: 1fr;
            grid-template-areas:
                "header"
                "main";
        }

        .sidenav-area {
            position: fixed;
            top: 3.5rem; /* Approximate header height */
            bottom: 0;
            left: 0;
            width: 250px;
            transform: translateX(-100%);
            grid-area: unset;
        }

        .sidenav-area.open {
            transform: translateX(0);
        }

        .menu-backdrop {
            position: fixed;
            inset: 0;
            background: rgba(0, 0, 0, 0.5);
            backdrop-filter: blur(4px);
            z-index: 30;
        }

        .page-container {
            padding: 1rem;
        }

        .main-area :global(.main-backdrop) {
            --box-border-radius: 0;
        }
    }
</style>
