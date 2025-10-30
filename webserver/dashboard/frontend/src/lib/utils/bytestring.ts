export const ByteUnits = {
    B: 1,
    K: 1024,
    M: 1024 * 1024,
    G: 1024 * 1024 * 1024,
    T: 1024 * 1024 * 1024 * 1024,
    P: 1024 * 1024 * 1024 * 1024 * 1024,
    E: 1024 * 1024 * 1024 * 1024 * 1024 * 1024,
    Z: 1024 * 1024 * 1024 * 1024 * 1024 * 1024 * 1024,
    Y: 1024 * 1024 * 1024 * 1024 * 1024 * 1024 * 1024 * 1024,
};

const unitLabels = Object.keys(ByteUnits);

// Format bytes into human-readable string using the largest unit possible
export function formatBytesToLargest(bytes: number, decimals = 2): string {
    if (bytes === 0) return "0B";

    const base = 1024;
    const dm = decimals < 0 ? 0 : decimals;

    const i = Math.floor(Math.log2(bytes) / 10);

    return parseFloat((bytes / Math.pow(base, i)).toFixed(dm)) + unitLabels[i];
}

// Format bytes into human-readable string using a fixed unit
export function formatBytes(bytes: number, unit: keyof typeof ByteUnits, decimals = 2): string {
    if (bytes === 0) return "0" + unitLabels[0];

    const dm = decimals < 0 ? 0 : decimals;
    return parseFloat((bytes / ByteUnits[unit]).toFixed(dm)) + unit;
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
