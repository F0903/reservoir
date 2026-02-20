export class Viewport {
    #isMobile = $state(false);

    constructor(cutoff: number = 768) {
        if (typeof window === "undefined") return;

        const query = window.matchMedia(`(max-width: ${cutoff}px)`);
        this.#isMobile = query.matches;

        query.addEventListener("change", (e) => {
            this.#isMobile = e.matches;
        });
    }

    get isMobile() {
        return this.#isMobile;
    }
}

export const viewport = new Viewport();
