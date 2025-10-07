// Treat functions and arrays specially; otherwise recurse through objects.
type AnyFunction = (..._args: unknown[]) => unknown;
type AnyArray<T = unknown> = Array<T>;
type AnyROArray<T = unknown> = ReadonlyArray<T>;

// Utility type for merge
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

function arraysEqual(
    a: unknown[],
    b: unknown[],
    compare: (_a: unknown, _b: unknown) => boolean = Object.is,
): boolean {
    if (a.length !== b.length) return false;
    for (let i = 0; i < a.length; i++) {
        if (!compare(a[i], b[i])) return false;
    }
    return true;
}

type Comparator = (_a: unknown, _b: unknown) => boolean;
type KeyTransform = (_key: string) => string;

export type MergeOptions = {
    keyTransform?: KeyTransform; // map source key -> target key
    recurse?: boolean; // default true
    replaceArrays?: boolean; // default true (otherwise shallow index-wise patch)
    allowNull?: boolean; // default true; if false, treat null like undefined (skip)
    compare?: Comparator; // default Object.is
};

// Applies changes from one object to another recursively
// Returns true if any changes were made
export function patch<T extends Record<string, unknown>>(
    to: T,
    from: DeepPartial<T>,
    keyTransformOrOptions?: KeyTransform | MergeOptions,
    recurseFlag?: boolean,
): boolean {
    const defaults: MergeOptions = {
        recurse: true,
        replaceArrays: false,
        allowNull: true,
        compare: Object.is,
    };
    const options: MergeOptions = { ...defaults };
    if (typeof keyTransformOrOptions === "function") {
        options.keyTransform = keyTransformOrOptions;
    } else if (keyTransformOrOptions && typeof keyTransformOrOptions === "object") {
        Object.assign(options, keyTransformOrOptions);
    }
    if (typeof recurseFlag === "boolean") options.recurse = recurseFlag;

    const compare = options.compare ?? Object.is;

    let changed = false;

    for (const key of Object.keys(from as object)) {
        // guard against prototype pollution
        if (key === "__proto__" || key === "constructor" || key === "prototype") continue;

        const fromKey = key; // original key in 'from'
        const toKey = options.keyTransform ? options.keyTransform(fromKey) : fromKey;
        const toRec = to as Record<string, unknown>;

        const bValue = (from as Record<string, unknown>)[fromKey];
        if (bValue === undefined) continue;
        if (bValue === null && !options.allowNull) continue;
        const bRec = bValue as Record<string, unknown>;

        const aValue = (to as Record<string, unknown>)[toKey];
        const aRec = aValue as Record<string, unknown>;

        // Recurse into plain objects when both sides are plain and recurse enabled
        if (options.recurse && isPlain(bValue) && isPlain(aValue)) {
            if (patch(aRec, bRec, options)) changed = true;
            continue;
        }

        // Arrays handling
        if (Array.isArray(bValue) && Array.isArray(aValue)) {
            if (options.replaceArrays) {
                if (!arraysEqual(aValue, bValue, compare)) {
                    toRec[toKey] = bValue;
                    changed = true;
                }
            } else {
                // index-wise shallow patch up to min length; replace if length differs
                const minLen = Math.min(aValue.length, bValue.length);
                let localChanged = aValue.length !== bValue.length;
                for (let i = 0; i < minLen; i++) {
                    if (!compare(aValue[i], bValue[i])) {
                        aValue[i] = bValue[i];
                        localChanged = true;
                    }
                }
                if (aValue.length > bValue.length) {
                    aValue.length = bValue.length; // truncate if longer
                    localChanged = true;
                } else if (aValue.length < bValue.length) {
                    // push remaining items from other if shorter
                    const diff = bValue.length - aValue.length;
                    for (let i = 0; i < diff; i++) {
                        aValue.push(bValue[minLen + i]);
                    }
                }

                if (localChanged) changed = true;
            }
            continue;
        }

        if (!compare(aValue, bValue)) {
            toRec[toKey] = bValue;
            changed = true;
        }
    }

    return changed;
}
