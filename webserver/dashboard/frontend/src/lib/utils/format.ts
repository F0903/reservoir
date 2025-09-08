export const ByteUnits = {
    B: 1,
    K: 1024,
    M: 1024 * 1024,
    G: 1024 * 1024 * 1024,
    T: 1024 * 1024 * 1024 * 1024,
};

// Format bytes into human-readable string
export function formatBytes(bytes: number, unit: keyof typeof ByteUnits, decimals = 2): string {
    if (bytes === 0) return "0B";

    return parseFloat((bytes / ByteUnits[unit]).toFixed(decimals)) + unit;
}

export function parseByteString(s: string): number {
    const unitRe = /^(\d+)([BKMGT])$/i;

    const matches = unitRe.exec(s);
    if (!matches) throw new Error(`Invalid byte string: ${s}`);

    const value = parseInt(matches[1]);
    const unit = matches[2].toUpperCase() as keyof typeof ByteUnits;

    if (!(unit in ByteUnits)) throw new Error(`Invalid byte unit: ${unit}`);

    return value * ByteUnits[unit];
}

export function snakeCaseToCamelCase(s: string): string {
    return s.replace(/_+([a-z0-9])/gi, (_m, c: string) => c.toUpperCase());
}
