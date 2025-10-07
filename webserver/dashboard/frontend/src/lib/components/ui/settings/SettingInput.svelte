<script lang="ts" generics="C extends Component<any, any, 'value'>, O">
    import { log } from "$lib/utils/logger";
    import { type Component, type ComponentProps } from "svelte";

    // Value type exposed by the InputComponent's `value` prop
    type V = ComponentProps<C>["value"];

    let {
        InputComponent,
        get,
        commit: commitValue,
        valueTransform,
        onChange,
        disabled = false,
        ...restProps
    }: {
        InputComponent: C;
        get: () => V;
        commit: (val: any) => Promise<any>;
        valueTransform?: (val: V) => O;
        onChange?: (different: boolean) => void;
        disabled?: boolean;
    } = $props();

    let store: V = $state(get());

    let value: V | undefined = $state();
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
        if (!hasChanged()) return;

        try {
            let valueToWrite = value;
            if (valueTransform) {
                valueToWrite = valueTransform(value);
                log.debug(`Committing setting with transformed value: ${valueToWrite}`);
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
        store = value;
    }
</script>

<InputComponent bind:value {disabled} {...restProps}></InputComponent>
