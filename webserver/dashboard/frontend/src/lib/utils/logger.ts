export const log = {
    debug: (...data: unknown[]) => {
        if (import.meta.env.DEV) {
            console.log(...data);
        }
    },
    error: (...data: unknown[]) => {
        console.error(...data);
    },
};
