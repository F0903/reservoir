// Deeply merges two objects, with the second object taking precedence
export function deepMerge(
    target: Record<string, unknown>,
    source: Record<string, unknown>,
): Record<string, unknown> {
    const result = { ...target };

    for (const key in source) {
        if (source[key] !== undefined) {
            if (isObject(source[key]) && isObject(result[key])) {
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

function isObject(item: unknown): item is Record<string, unknown> {
    return item !== null && typeof item === "object" && !Array.isArray(item);
}
