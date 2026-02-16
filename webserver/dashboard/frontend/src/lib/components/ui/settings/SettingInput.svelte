<script
    lang="ts"
    generics="C extends Component<CP, CE, 'value'>, CP extends { label: string, value: string | number | boolean }, CE extends Record<string, unknown> = Record<string, unknown>, V = CP['value'], O = V"
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

    // svelte-ignore state_referenced_locally
    let inputValue: V = $state(get());

    // Has 'inputValue' changed from 'get()'?
    export function hasDiverged() {
        if (get() === undefined) return false;
        log.debug(`Checking divergence: inputValue=${inputValue}, get()=${get()}`);
        // We use != to allow type coercion (e.g. between number and string)
        return inputValue != get();
    }

    export async function commit() {
        if (!hasDiverged() || inputValue === undefined) return;

        try {
            let valueToWrite: O;
            if (valueTransform) {
                valueToWrite = valueTransform(inputValue);
                log.debug(`Committing setting with transformed value: ${valueToWrite}`);
            } else {
                valueToWrite = inputValue as O;
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
        inputValue = get();
    }

    function getValue() {
        return inputValue;
    }

    function setValue(newValue: V) {
        inputValue = newValue;
        onChange?.(hasDiverged());
    }
</script>

<InputComponent {...restProps as CP} bind:value={getValue, setValue} />
