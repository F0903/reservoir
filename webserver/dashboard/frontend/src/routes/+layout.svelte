<script>
    import "../global.css";
    import "@fontsource-variable/open-sans";
    import Header from "$lib/components/layout/Header.svelte";
    import SideNav from "$lib/components/layout/SideNav.svelte";
    import BackdropBox from "$lib/components/ui/BackdropBox.svelte";
    import SideNavButton from "$lib/components/layout/SideNavButton.svelte";
    import { LayoutDashboard, Settings } from "@lucide/svelte";

    let { children } = $props();
</script>

<div class="layout-grid">
    <div class="header-area">
        <Header />
    </div>
    <div class="sidenav-area">
        <SideNav>
            <SideNavButton url="/"><LayoutDashboard />Dashboard</SideNavButton>
            <SideNavButton url="/settings"><Settings />Settings</SideNavButton>
        </SideNav>
    </div>
    <div class="main-area">
        <BackdropBox
            --box-backdrop-color="var(--quaternary-color)"
            --box-border-radius="25px 0px 0px 0px"
            --box-shadow="5px 5px 15px 2px var(--default-shadow)"
        >
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
    }

    .sidenav-area {
        grid-area: sidenav;
    }

    .main-area {
        grid-area: main;
    }

    .page-container {
        overflow-y: auto;
        padding: 2rem;
    }
</style>
