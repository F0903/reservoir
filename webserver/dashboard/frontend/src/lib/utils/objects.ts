import { log } from "./logger";

function isPlain(o: unknown): o is Record<string, unknown> {
    return o != null && Object.getPrototypeOf(o) === Object.prototype;
}

function arraysEqual(a: unknown[], b: unknown[]) {
    if (a.length !== b.length) return false;

    for (let i = 0; i < a.length; i++) {
        if (!Object.is(a[i], b[i])) return false;
    }
    return true;
}

// Deeply merges two objects, with the second object taking precedence
export function deepMerge(
    target: Record<string, unknown>,
    source: Record<string, unknown>,
): Record<string, unknown> {
    const result = { ...target };

    for (const key in source) {
        if (source[key] !== undefined) {
            if (isPlain(source[key]) && isPlain(result[key])) {
                result[key] = deepMerge(
                    result[key] as Record<string, unknown>,
                    source[key] as Record<string, unknown>,
                );
            } else {
                result[key] = source[key];
            }
        }
    }

    return result;
}

// Applies changes from object B to object A
export function patch<A extends Record<string, unknown>>(
    a: A,
    b: Partial<A>,
    keyTransform?: (_key: string) => string,
    recurse: boolean = true,
): boolean {
    let changed = false;
    for (const key in b) {
        if (!Object.prototype.hasOwnProperty.call(b, key)) continue;

        const keyStr = keyTransform ? keyTransform(key) : key;
        const bValue = b[keyStr as keyof A];
        const aValue = a[keyStr as keyof A];

        if (recurse && isPlain(bValue) && isPlain(aValue)) {
            if (patch(aValue as Record<string, unknown>, bValue as Record<string, unknown>)) {
                changed = true;
            }
        } else if (Array.isArray(bValue) && Array.isArray(aValue)) {
            if (!arraysEqual(aValue, bValue)) {
                (a as Record<string, unknown>)[keyStr] = bValue;
                changed = true;
            }
        } else if (!Object.is(aValue, bValue)) {
            (a as Record<string, unknown>)[keyStr] = bValue;
            changed = true;
        }
    }
    return changed;
}

export function setPropIfChanged<T>(
    name: string,
    object: Record<string, T>,
    currentValue: T,
    setter: (_value: T) => void,
) {
    if (!object || !(name in object)) {
        log.debug(`Property '${name}' not found in object, not setting.`);
        return;
    }

    const newValue = object[name];
    if (newValue === currentValue) {
        log.debug(`Property '${name}' has not changed. Not setting...`);
        return;
    }

    log.debug(`Set prop '${name}' to ${newValue}. Was ${currentValue}`);
    setter(newValue);
}

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
