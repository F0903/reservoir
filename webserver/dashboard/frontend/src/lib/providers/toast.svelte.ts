import Toast, { type ToastProps } from "$lib/components/ui/Toast.svelte";
import { mount, unmount } from "svelte";

export class ToastHandle {
    #closed: boolean = false;
    #closer: () => void;

    constructor(closer: () => void, closed = false) {
        this.#closer = closer;
        this.#closed = closed;
    }

    close = () => {
        if (this.#closed) return;
        this.#closed = true;
        this.#closer();
    };
}

export class ToastProvider {
    // Show a new toast and return a function to close it
    show = (props: ToastProps): ToastHandle => {
        let toast: Toast | null = null;
        const handle = new ToastHandle(() => {
            if (toast) this.#closeToast(toast);
        });

        toast = mount(Toast, {
            target: document.body,
            props: {
                handle,
                ...props,
            },
        });
        return handle;
    };

    #closeToast = (toast: Toast) => {
        // We don't care about waiting for the transition to finish, so no await.
        unmount(toast, {
            outro: true,
        });
    };
}
