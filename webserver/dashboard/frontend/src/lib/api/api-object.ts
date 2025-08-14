export type FetchFn = (_input: RequestInfo | URL, _init?: RequestInit) => Promise<Response>;

export interface APIObjectConstructor<T> {
    new (_json: Record<string, unknown>): T;
}

// Represents a raw JSON API object.
export class RawAPIObject {
    [key: string]: unknown;

    constructor(json: Record<string, unknown>) {
        Object.assign(this, json);
    }
}

export async function apiGet<T>(
    endpoint: string,
    type: APIObjectConstructor<T>,
    fetchFn: FetchFn = fetch,
): Promise<T> {
    const fullEndpoint = `/api${endpoint}`;

    // We use window.location.origin since the frontend is served embedded from
    // a webserver in the proxy itself, so the backend API is from the same origin.
    const base = window.location.origin;
    const url = new URL(fullEndpoint, base);

    const response = await fetchFn(url);
    if (!response.ok) {
        throw new Error(`Failed to fetch from '${url}'`);
    }

    const json = await response.json();

    return new type(json) as T;
}
