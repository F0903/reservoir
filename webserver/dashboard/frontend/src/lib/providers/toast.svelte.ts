import Toast, { type ToastProps } from "$lib/components/ui/Toast.svelte";
import { mount, unmount } from "svelte";

export class ToastProvider {
    private openToast: Toast | null = null;

    show = (props: ToastProps) => {
        if (this.openToast) return; // As of now, only one toast can be open at a time

        this.openToast = mount(Toast, {
            target: document.body,
            props: {
                ...props,
            },
        });
    };

    close = () => {
        if (!this.openToast) return;

        unmount(this.openToast, {
            outro: true,
        });
        this.openToast = null;
    };
}
