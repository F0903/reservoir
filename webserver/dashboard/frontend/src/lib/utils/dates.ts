export function formatTimeSinceDate(start: Date) {
    const now = new Date();
    const diffInMs = now.getTime() - start.getTime();

    const msPerSecond = 1000;
    const msPerMinute = msPerSecond * 60;
    const msPerHour = msPerMinute * 60;
    const msPerDay = msPerHour * 24;

    const days = Math.floor(diffInMs / msPerDay);
    const hours = Math.floor((diffInMs % msPerDay) / msPerHour);
    const minutes = Math.floor((diffInMs % msPerHour) / msPerMinute);
    const seconds = Math.floor((diffInMs % msPerMinute) / msPerSecond);

    if (days > 0) {
        return `${days}d ${hours}h ${minutes}m`;
    } else if (hours > 0) {
        return `${hours}h ${minutes}m ${seconds}s`;
    } else if (minutes > 0) {
        return `${minutes}m ${seconds}s`;
    } else {
        return `${seconds}s`;
    }
}
