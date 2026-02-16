export function isDebugMode(): boolean {
    return import.meta.env.MODE === "development";
}

export const log = {
    debug: (...data: unknown[]) => {
        if (isDebugMode()) {
            console.log(...data);
        }
    },
    error: (...data: unknown[]) => {
        console.error(...data);
    },
};
