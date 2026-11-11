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

    // Rates can be below one byte, or negative: both must still land on a real unit.
    const i = Math.max(0, Math.floor(Math.log2(Math.abs(bytes)) / 10));

    return parseFloat((bytes / Math.pow(base, i)).toFixed(dm)) + unitLabels[i];
}
