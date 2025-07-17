<script>
    import "../global.css";
    import Header from "$lib/components/layout/Header.svelte";
    import SideNav from "$lib/components/layout/SideNav.svelte";
    import RoundedCornerBox from "$lib/components/ui/RoundedCornerBox.svelte";

    let { children } = $props();
</script>

<div class="layout-grid">
    <div class="header-area">
        <Header />
    </div>
    <div class="sidenav-area">
        <SideNav><p>hello</p></SideNav>
    </div>
    <div class="main-area">
        <RoundedCornerBox
            --box-border-radius="25px 0px 0px 0px"
            --box-shadow="5px 5px 15px 2px var(--default-shadow)"
        >
            <main>
                {@render children()}
            </main>
        </RoundedCornerBox>
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

    main {
        overflow-y: auto;
        padding: 1rem;
    }
</style>
