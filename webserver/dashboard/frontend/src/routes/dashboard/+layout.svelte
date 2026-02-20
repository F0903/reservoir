<script lang="ts">
    import Header from "$lib/components/layout/Header.svelte";
    import SideNav from "$lib/components/layout/SideNav.svelte";
    import BackdropBox from "$lib/components/ui/BackdropBox.svelte";
    import SideNavButton from "$lib/components/layout/SideNavButton.svelte";
    import { LayoutDashboard, Logs, Settings } from "@lucide/svelte";

    let { children } = $props();

    let isMenuOpen = $state(false);

    function toggleMenu() {
        isMenuOpen = !isMenuOpen;
    }

    function closeMenu() {
        isMenuOpen = false;
    }
</script>

<div class="layout-grid">
    <div class="header-area">
        <Header onToggleMenu={toggleMenu} />
    </div>
    <div class="sidenav-area" class:open={isMenuOpen}>
        <SideNav>
            <SideNavButton url="/dashboard" onClick={closeMenu}>
                <LayoutDashboard />Dashboard
            </SideNavButton>
            <SideNavButton url="/dashboard/settings" onClick={closeMenu}>
                <Settings />Settings
            </SideNavButton>
            <SideNavButton url="/dashboard/log" onClick={closeMenu}>
                <Logs />Log
            </SideNavButton>
        </SideNav>
    </div>

    {#if isMenuOpen}
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
