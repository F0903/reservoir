import { log } from "./logger";

export function readPropOrDefault<T>(
    name: string,
    object: Record<string, unknown>,
    defaultValue: T,
): T {
    if (!object || !(name in object)) {
        log.debug(`Property '${name}' not found in object, returning default value.`);
        return defaultValue;
    }

    return object[name] as T;
}

export function getPropAssert<T>(name: string, object: Record<string, unknown>): T {
    const value = object[name];

    if (value === undefined) {
        throw new Error(`Property '${name}' is missing from object.`);
    }
    return value as T;
}
