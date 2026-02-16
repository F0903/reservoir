import type { ToastProps } from "$lib/components/ui/Toast.svelte";
import { log } from "$lib/utils/logger";

export interface ToastEntry {
    id: string;
    props: ToastProps;
    handle: ToastHandle;
}

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
    // List of active toasts
    toasts = $state<ToastEntry[]>([]);

    // Show a new toast and return a handle to close it
    show = (props: ToastProps): ToastHandle => {
        const id = Math.random().toString(36).substring(2, 9);
        const handle = new ToastHandle(() => this.closeToast(id));

        log.debug("Adding toast:", id, props.message);
        this.toasts.push({ id, props, handle });

        return handle;
    };

    // Show a new info toast and return a handle to close it
    info = (message: string, durationMs = 5000, options: Partial<ToastProps> = {}): ToastHandle => {
        return this.show({
            type: "info",
            message,
            durationMs,
            ...options,
        } as ToastProps);
    };

    // Show a new success toast and return a handle to close it
    success = (
        message: string,
        durationMs = 3000,
        options: Partial<ToastProps> = {},
    ): ToastHandle => {
        return this.show({
            type: "success",
            message,
            durationMs,
            ...options,
        } as ToastProps);
    };

    // Show a new error toast and return a handle to close it
    error = (
        message: string,
        durationMs = 10000,
        options: Partial<ToastProps> = {},
    ): ToastHandle => {
        return this.show({
            type: "error",
            message,
            durationMs,
            ...options,
        } as ToastProps);
    };

    // Show a new action toast and return a handle to close it
    action = (
        message: string,
        options: Omit<ToastProps & { type: "action" }, "type" | "message">,
    ): ToastHandle => {
        return this.show({
            type: "action",
            message,
            ...options,
        } as ToastProps);
    };

    private closeToast = (id: string) => {
        const index = this.toasts.findIndex((t) => t.id === id);
        if (index !== -1) {
            log.debug("Removing toast:", id);
            this.toasts.splice(index, 1);
        }
    };
}
