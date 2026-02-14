import Toast, { type ToastProps } from "$lib/components/ui/Toast.svelte";
import { log } from "$lib/utils/logger";
import { mount, unmount } from "svelte";

// A class that provides a "handle" to a toast that can be used to close it.
export class ToastHandle {
    private closed: boolean = false;
    private closer: () => void;

    constructor(closer: () => void, closed = false) {
        this.closer = closer;
        this.closed = closed;
    }

    close = () => {
        if (this.closed) {
            log.debug("Toast already closed, ignoring close() call");
            return;
        }
        log.debug("Closing toast...");
        this.closed = true;
        this.closer();
    };
}

export class ToastProvider {
    private target: HTMLElement;

    constructor(target: HTMLElement = document.body) {
        this.target = target;
    }

    // Show a new toast and return a function to close it
    show = (props: ToastProps): ToastHandle => {
        let toast: Toast | null = null;
        const handle = new ToastHandle(() => {
            if (toast) this.closeToast(toast);
        });

        toast = mount(Toast, {
            target: this.target,
            props: {
                handle,
                ...props,
            },
        });
        return handle;
    };

    private closeToast = (toast: Toast) => {
        // We don't care about waiting for the transition to finish, so no await.
        unmount(toast, {
            outro: true,
        });
    };
}
