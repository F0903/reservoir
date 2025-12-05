<script
    lang="ts"
    generics="C extends Component<CP, CE, 'value'>, CP extends { value: unknown }, CE extends Record<string, unknown> = Record<string, unknown>, V = CP['value'], O = V"
>
    import { log } from "$lib/utils/logger";
    import type { Component } from "svelte";

    let {
        InputComponent,
        get,
        commit: commitValue,
        valueTransform,
        onChange,
        ...restProps
    }: {
        InputComponent: C;
        get: () => V;
        commit: (_val: O) => Promise<unknown>;
        valueTransform?: (_val: V) => O;
        onChange?: (_different: boolean) => void;
        [key: string]: unknown;
    } = $props();

    // We want the behaviour the warning warns about.
    // svelte-ignore state_referenced_locally
    let value: V = $state(get());
    let store: V = $derived(value);
    let lastChangeValue = false;

    $effect(() => {
        const change = hasChanged();
        if (change !== lastChangeValue) {
            lastChangeValue = change;
            log.debug(`SettingInput change state updated: ${change}`);
            onChange?.(change);
        }
    });

    // Has 'value' changed from 'store'?
    export function hasChanged() {
        // We use != to allow type coercion (e.g. between number and string)
        return value != store;
    }

    export async function commit() {
        if (!hasChanged() || value === undefined) return;

        try {
            let valueToWrite: O;
            if (valueTransform) {
                valueToWrite = valueTransform(value);
                log.debug(`Committing setting with transformed value: ${valueToWrite}`);
            } else {
                valueToWrite = value as O;
            }
            await commitValue(valueToWrite);

            log.debug("Setting committed successfully.");
        } catch (e) {
            log.error("Failed to commit setting:", e);

            // Error toast will be shown by global handler.
            throw e;
        }
    }

    export async function reset() {
        value = await get();
    }
</script>

<InputComponent {...restProps as CP} bind:value />
