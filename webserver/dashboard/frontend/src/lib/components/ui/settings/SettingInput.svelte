<script lang="ts" generics="C extends Component<any, any, 'value'>, O">
    import { log } from "$lib/utils/logger";
    import { onMount, type Component, type ComponentProps } from "svelte";

    // Value type exposed by the InputComponent's `value` prop
    type V = ComponentProps<C>["value"];

    type SettingV = string | number | boolean;

    let {
        InputComponent,
        getSetting,
        setSetting,
        settingTransform,
        onChange,
        disabled = false,
        ...restProps
    }: {
        InputComponent: C;
        getSetting: () => Promise<SettingV> | SettingV;
        setSetting: (_value: O) => any;
        settingTransform: (val: V) => O;
        onChange?: (different: boolean) => void;
        disabled?: boolean;
    } = $props();

    let startValue: V | undefined;
    let value: V | undefined = $state();

    onMount(async () => {
        await reset(); // Fetch and set the value and startValue on mount
    });

    $effect(() => {
        const changed = hasChanged();
        onChange?.(changed);
    });

    export function hasChanged() {
        // We use != to allow type coercion (e.g. between number and string)
        return value != startValue;
    }

    export async function save() {
        if (!hasChanged()) return;

        try {
            // setSetting might be async, so we await it.
            await setSetting(settingTransform(value));
        } catch (e) {
            log.error("Failed to save setting:", e);
            // Error toast will be shown by global handler.
            throw e;
        }
    }

    export async function reset() {
        value = await getSetting();
        startValue = value;
    }
</script>

<InputComponent bind:value {disabled} {...restProps}></InputComponent>
