import { get, type Writable } from "svelte/store";
import { getPropAssert } from "./values";

export function isStore<T>(value: unknown): value is Writable<T> {
    return (
        value != null &&
        typeof value === "object" &&
        "subscribe" in value &&
        typeof value.subscribe === "function"
    );
}

export function getPropStore<T>(name: string, object: Record<string, unknown>): T {
    const prop = getPropAssert(name, object);
    if (isStore(prop)) {
        return get(prop) as T;
    } else {
        return prop as T;
    }
}

export function setPropStore(name: string, object: Record<string, unknown>, value: unknown): void {
    const prop = getPropAssert(name, object);
    if (isStore(prop)) {
        prop.set(value);
    } else {
        object[name] = value;
    }
}
