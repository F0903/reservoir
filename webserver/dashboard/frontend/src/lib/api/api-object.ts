import type { JSONResponse } from "$lib/utils/json";

export type FetchFn = (_input: RequestInfo | URL, _init?: RequestInit) => Promise<Response>;

export interface APIObjectType<T> {
    new (_json: JSONResponse): T;
}

export async function apiGet<T>(
    endpoint: string,
    type: APIObjectType<T>,
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
