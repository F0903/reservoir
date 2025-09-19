import { log } from "../logger";

export function setPropIfChanged<T>(
    name: string,
    object: Record<string, T>,
    currentValue: T,
    setter: (_value: T) => void,
    comparer?: (_a: T, _b: T) => boolean,
) {
    if (!object || !(name in object)) {
        log.debug(`Property '${name}' not found in object, not setting.`);
        return;
    }

    const newValue = object[name];
    const equals = comparer ? comparer(newValue, currentValue) : Object.is(newValue, currentValue);
    if (equals) {
        log.debug(`Property '${name}' has not changed. Not setting...`);
        return;
    }

    log.debug(`Set prop '${name}' to ${String(newValue)}. Was ${String(currentValue)}`);
    setter(newValue);
}
