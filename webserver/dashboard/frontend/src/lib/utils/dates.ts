export function formatTimeSinceDate(start: Date) {
    const diff = new Date(Date.now() - start.getTime());

    const years = diff.getUTCFullYear() - 1970;
    const months = diff.getUTCMonth();
    const days = diff.getUTCDate() - 1;
    const hours = diff.getUTCHours();
    const minutes = diff.getUTCMinutes();
    const seconds = diff.getUTCSeconds();

    const parts = [];
    if (years) parts.push(`${years}y`);
    if (months) parts.push(`${months}m`);
    if (days) parts.push(`${days}d`);
    if (hours) parts.push(`${hours}h`);
    if (minutes) parts.push(`${minutes}m`);
    if (seconds || parts.length === 0) parts.push(`${seconds}s`);

    return parts.slice(0, 3).join(" ");
}
