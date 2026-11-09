import { log } from "./logger";

// Treat functions and arrays specially; otherwise recurse through objects.
type AnyFunction = (..._args: unknown[]) => unknown;
type AnyArray<T = unknown> = Array<T>;
type AnyROArray<T = unknown> = ReadonlyArray<T>;

// Utility type for merge/patch
export type DeepPartial<T> = T extends AnyFunction
    ? T
    : T extends AnyROArray<infer U>
      ? ReadonlyArray<DeepPartial<U>>
      : T extends AnyArray<infer U>
        ? AnyArray<DeepPartial<U>>
        : T extends object
          ? { [K in keyof T]?: DeepPartial<T[K]> }
          : T;

// Check if value is a plain object (not null, not array, not class instance)
function isPlain(o: unknown): o is Record<string, unknown> {
    return o != null && Object.getPrototypeOf(o) === Object.prototype;
}

type Comparator = (_a: unknown, _b: unknown) => boolean;

export type MergeOptions = {
    recurse?: boolean; // default true
    allowNull?: boolean; // default true; if false, treat null like undefined (skip)
    compare?: Comparator; // default Object.is
};

// Applies changes from one object to another recursively
// Returns true if any changes were made
export function patch<T extends Record<string, unknown>>(
    to: T,
    from: DeepPartial<T>,
    mergeOptions?: MergeOptions,
    recurseFlag?: boolean,
): boolean {
    const defaults: MergeOptions = {
        recurse: true,
        allowNull: true,
        compare: Object.is,
    };
    const options: MergeOptions = { ...defaults, ...mergeOptions };
    if (typeof recurseFlag === "boolean") options.recurse = recurseFlag;

    const compare = options.compare ?? Object.is;

    let changed = false;

    for (const key of Object.keys(from as object)) {
        // guard against prototype pollution
        if (key === "__proto__" || key === "constructor" || key === "prototype") continue;

        const toRec = to as Record<string, unknown>;

        const bValue = (from as Record<string, unknown>)[key];
        if (bValue === undefined) continue;
        if (bValue === null && !options.allowNull) continue;
        const bRec = bValue as Record<string, unknown>;

        const aValue = (to as Record<string, unknown>)[key];
        const aRec = aValue as Record<string, unknown>;

        // Recurse into plain objects when both sides are plain and recurse enabled
        if (options.recurse && isPlain(bValue) && isPlain(aValue)) {
            if (patch(aRec, bRec, options)) {
                changed = true;
                log.debug(`(patch) Recursed patch returned changes at key ${key}`);
            }
            continue;
        }

        // Arrays handling
        if (Array.isArray(bValue) && Array.isArray(aValue)) {
            // index-wise shallow patch up to min length; replace if length differs
            const minLen = Math.min(aValue.length, bValue.length);
            let localChanged = aValue.length !== bValue.length;
            for (let i = 0; i < minLen; i++) {
                if (!compare(aValue[i], bValue[i])) {
                    aValue[i] = bValue[i];
                    localChanged = true;
                    log.debug(`(patch) Patched array item at key ${key}[${i}]`);
                }
            }
            if (aValue.length > bValue.length) {
                aValue.length = bValue.length; // truncate if longer
            } else if (aValue.length < bValue.length) {
                // push remaining items from other if shorter
                const diff = bValue.length - aValue.length;
                for (let i = 0; i < diff; i++) {
                    aValue.push(bValue[minLen + i]);
                    log.debug(`(patch) Pushed array item at key ${key}[${minLen + i}]`);
                }
            }

            if (localChanged) {
                changed = true;
                log.debug(`(patch) Making array as changed...`);
            }
            continue;
        }

        if (!compare(aValue, bValue)) {
            toRec[key] = bValue;
            changed = true;
            log.debug(`(patch) Updated key ${key}`);
        }
    }

    log.debug(`(patch) Finished patching, changed=${changed}`);
    return changed;
}
