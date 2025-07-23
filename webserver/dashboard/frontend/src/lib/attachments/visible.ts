import type { Attachment } from "svelte/attachments";

export function visible(callback: (_isVisible: boolean) => void): Attachment<Element> {
    return (element: Element) => {
        const observer = new IntersectionObserver(
            (entries) => {
                entries.forEach((entry) => {
                    callback(entry.isIntersecting);
                });
            },
            { threshold: 0.1 },
        );

        observer.observe(element);

        return () => {
            observer.disconnect();
        };
    };
}
