import { doIfDebug } from "./conditional";

export const log = {
    debug: (...data: unknown[]) => {
        doIfDebug(() => {
            console.log(...data);
        });
    },
    error: (...data: unknown[]) => {
        console.error(...data);
    },
};
