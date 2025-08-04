import { log } from "./logger";

export interface JSONResponse {
    [key: string]: unknown;
}

export function readPropOrDefault<T>(name: string, json: JSONResponse, defaultValue: T): T {
    if (!json || !(name in json)) {
        log.debug(`Property '${name}' not found in JSON response, returning default value.`);
        return defaultValue;
    }

    return json[name] as T;
}
