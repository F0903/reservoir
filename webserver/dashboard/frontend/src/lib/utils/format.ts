// Format bytes into human-readable string
export function formatBytes(bytes: number, decimals = 2): string {
    if (bytes === 0) return "0 Bytes";

    const k = 1024;
    const dm = decimals < 0 ? 0 : decimals;
    const sizes = ["Bytes", "KB", "MB", "GB", "TB", "PB", "EB", "ZB", "YB"];

    const i = Math.floor(Math.log(bytes) / Math.log(k));

    return parseFloat((bytes / Math.pow(k, i)).toFixed(dm)) + " " + sizes[i];
}

// Format number with thousand separators
export function formatNumber(num: number): string {
    return num.toLocaleString();
}

export function formatPercentage(value: number, total: number, decimals = 1): string {
    if (total === 0) return "0%";
    return ((value / total) * 100).toFixed(decimals) + "%";
}

export function snakeCaseToCamelCase(s: string): string {
    return s.replace(/_+([a-z0-9])/gi, (_m, c: string) => c.toUpperCase());
}
