import { browser } from "$app/environment";

// Only run the function if we are in debug mode
export function doIfDebug(fn: () => void): void {
    if (import.meta.env.MODE === "development") {
        fn();
    }
}

// Only run the function if we are in a browser environment
export function doIfBrowser(fn: () => void): void {
    if (browser) {
        fn();
    }
}
