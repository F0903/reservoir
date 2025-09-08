<script lang="ts" generics="T extends Component<any, any, keyof {value: string}>">
    import type { Component, ComponentProps } from "svelte";

    // We handle these props in here
    type OmittedProps = "value" | "placeholder" | "onSubmit";

    let {
        getSetting,
        setSetting,
        settingTransform,
        onChange,
        InputComponent,
        ...restProps
    }: {
        getSetting: () => Promise<any> | any;
        setSetting: (_value: any) => Promise<any> | any;
        settingTransform: (val: string) => any;
        onChange?: (different: boolean) => void;
        InputComponent: T;
    } & Omit<ComponentProps<T>, OmittedProps> = $props();

    let value = $state("");
    let placeholder = $state("");

    const phState = getSetting();
    if (phState instanceof Promise) {
        phState.then((v) => (placeholder = v));
    } else {
        placeholder = phState;
    }

    $effect(() => {
        const changed = hasChanged();
        onChange?.(changed);
    });

    export function hasChanged() {
        return value !== "";
    }

    export async function save() {
        if (!hasChanged()) return;
        await setSetting(settingTransform(value));
    }

    export async function reset() {
        value = "";
        placeholder = await getSetting();
    }
</script>

<InputComponent bind:value {placeholder} {...restProps}></InputComponent>
