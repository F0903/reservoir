import { browser } from "$app/environment";

// Only run the function if we are in debug mode
export function doDebug(fn: () => void): void {
    if (import.meta.env.MODE === "development") {
        fn();
    }
}

// Only run the function if we are in a browser environment
export function doBrowser(fn: () => void): void {
    if (browser) {
        fn();
    }
}
