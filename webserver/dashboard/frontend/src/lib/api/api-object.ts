export type FetchFn = (_input: RequestInfo | URL, _init?: RequestInit) => Promise<Response>;

export interface APIObjectConstructor<T> {
    new (_json: Record<string, unknown>): T;
}

// Represents a raw JSON API object.
export class APIJsonObject {
    [key: string]: unknown;

    constructor(json: Record<string, unknown>) {
        Object.assign(this, json);
    }
}

async function getAssert(endpoint: string, fetchFn: FetchFn = fetch): Promise<Response> {
    const fullEndpoint = `/api${endpoint}`;

    // We use window.location.origin since the frontend is served embedded from
    // a webserver in the proxy itself, so the backend API is from the same origin.
    const base = window.location.origin;
    const url = new URL(fullEndpoint, base);

    const response = await fetchFn(url);
    if (!response.ok) {
        throw new Error(`Failed to fetch from '${url}'`);
    }

    return response;
}

export async function apiGetTextStream(
    endpoint: string,
    fetchFn: FetchFn = fetch,
): Promise<ReadableStream<string>> {
    const resp = await getAssert(endpoint, fetchFn);
    if (!resp.body) {
        throw new Error(`Body was empty when fetching text stream from '${endpoint}'`);
    }

    return resp.body.pipeThrough(new TextDecoderStream());
}

export async function apiGet<T>(
    endpoint: string,
    type: APIObjectConstructor<T>,
    fetchFn: FetchFn = fetch,
): Promise<T> {
    const resp = await getAssert(endpoint, fetchFn);
    const json = await resp.json();

    return new type(json) as T;
}

export async function apiPatch(
    endpoint: string,
    json: Record<string, unknown>,
    fetchFn: FetchFn = fetch,
): Promise<void> {
    const response = await fetchFn(`/api${endpoint}`, {
        method: "PATCH",
        headers: {
            "Content-Type": "application/json",
        },
        body: JSON.stringify(json),
    });

    if (!response.ok) {
        throw new Error(`PATCH failed: ${response.status} ${response.statusText}`);
    }
}
