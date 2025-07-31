import { doDebug } from "./conditional";

export const log = {
    debug: (...data: unknown[]) => {
        doDebug(() => {
            console.log(...data);
        });
    },
    error: (...data: unknown[]) => {
        console.error(...data);
    },
};
