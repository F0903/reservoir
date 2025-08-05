<script lang="ts" generics="T extends Component<any, any, keyof {value: string}>">
    import { getPropStore, setPropStore } from "$lib/utils/stores";
    import type { Component, ComponentProps } from "svelte";

    // We handle these props in here
    type OmittedProps = "value" | "placeholder" | "onSubmit";

    let {
        settingName,
        settingObject,
        onChange,
        InputComponent,
        ...restProps
    }: {
        settingName: string;
        settingObject: any;
        onChange?: (different: boolean) => void;
        InputComponent: T;
    } & Omit<ComponentProps<T>, OmittedProps> = $props();

    let value = $state("");
    let placeholder = $state(getPropStore(settingName, settingObject));

    $effect(() => {
        const changed = hasChanged();
        onChange?.(changed);
    });

    export function hasChanged() {
        return value !== "";
    }

    export function save() {
        setPropStore(settingName, settingObject, value);
        reset();
    }

    export function reset() {
        value = "";
        placeholder = getPropStore(settingName, settingObject);
    }
</script>

<InputComponent bind:value {placeholder} {...restProps}></InputComponent>
