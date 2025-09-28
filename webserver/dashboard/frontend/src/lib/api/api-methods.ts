import { goto } from "$app/navigation";
import { resolve } from "$app/paths";
import { log } from "$lib/utils/logger";

export type FetchFn = (_input: RequestInfo | URL, _init?: RequestInit) => Promise<Response>;

export class UnauthorizedError extends Error {
    constructor() {
        super("Unauthorized, redirecting to login.");
        this.name = "UnauthorizedError";
    }
}

export type LoginRedirectOptions = { returnToLastWindow: boolean };

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

async function redirectToLogin(redirect: LoginRedirectOptions): Promise<void> {
    const params = new URLSearchParams();
    if (redirect.returnToLastWindow) {
        params.append("return", window.location.pathname + window.location.search);
    }
    let loginUrl = resolve(`/login`);
    if (params.size > 0) {
        loginUrl += `?${params.toString()}`;
    }
    await goto(loginUrl, {
        replaceState: true,
    });
}

async function assertResponse(
    response: Response,
    redirectOnUnauthorized: LoginRedirectOptions | null,
): Promise<void> {
    if (response.status === 401) {
        if (redirectOnUnauthorized) {
            await redirectToLogin(redirectOnUnauthorized);
            // We still continue to throw no matter what, so that the caller can handle it if they want to.
        }

        throw new UnauthorizedError();
    } else if (!response.ok) {
        throw new Error(
            `Failed to fetch from '${response.url}': ${response.status} ${response.statusText}`,
        );
    }
}

async function getAssert(
    endpoint: string,
    fetchFn: FetchFn = fetch,
    redirectOnUnauthorized: LoginRedirectOptions | null = { returnToLastWindow: true },
): Promise<Response> {
    const fullEndpoint = `/api${endpoint}`;

    // We use window.location.origin since the frontend is served embedded from
    // a webserver in the proxy itself, so the backend API is from the same origin.
    const base = window.location.origin;
    const url = new URL(fullEndpoint, base);

    const response = await fetchFn(url, {
        method: "GET",
        credentials: "same-origin",
    });
    await assertResponse(response, redirectOnUnauthorized);

    return response;
}

export async function apiGetTextStream(
    endpoint: string,
    fetchFn: FetchFn = fetch,
    redirectOnUnauthorized: LoginRedirectOptions | null = { returnToLastWindow: true },
): Promise<ReadableStream<string>> {
    try {
        const resp = await getAssert(endpoint, fetchFn);
        if (!resp.body) {
            throw new Error(`Body was empty when fetching text stream from '${endpoint}'`);
        }
        return resp.body.pipeThrough(new TextDecoderStream());
    } catch (err) {
        if (redirectOnUnauthorized && err instanceof UnauthorizedError) {
            return new ReadableStream<string>();
        } else {
            throw err;
        }
    }
}

export async function apiGet<T>(
    endpoint: string,
    type: APIObjectConstructor<T>,
    fetchFn: FetchFn = fetch,
    redirectOnUnauthorized: LoginRedirectOptions | null = { returnToLastWindow: true },
): Promise<T> {
    try {
        const resp = await getAssert(endpoint, fetchFn);
        const json = await resp.json();
        return new type(json) as T;
    } catch (err) {
        if (redirectOnUnauthorized && err instanceof UnauthorizedError) {
            return new type({}) as T;
        } else {
            throw err;
        }
    }
}

export async function apiPatch<T>(
    endpoint: string,
    json: Record<string, unknown>,
    fetchFn: FetchFn = fetch,
    redirectOnUnauthorized: LoginRedirectOptions | null = { returnToLastWindow: true },
): Promise<T> {
    const response = await fetchFn(`/api${endpoint}`, {
        method: "PATCH",
        headers: {
            "Content-Type": "application/json",
        },
        credentials: "same-origin",
        body: JSON.stringify(json),
    });
    await assertResponse(response, redirectOnUnauthorized);

    const contentType = response.headers.get("Content-Type");
    if (contentType && contentType.includes("application/json")) {
        log.debug("Parsing JSON response from PATCH");
        const respJson = await response.json();
        return respJson as T;
    }

    log.debug("Unknown or no content type in response from PATCH, returning body text");
    const respText = await response.text();
    return respText as T;
}

export async function apiPost<T>(
    endpoint: string,
    json: Record<string, unknown>,
    fetchFn: FetchFn = fetch,
    redirectOnUnauthorized: LoginRedirectOptions | null = { returnToLastWindow: true },
): Promise<T> {
    const response = await fetchFn(`/api${endpoint}`, {
        method: "POST",
        headers: {
            "Content-Type": "application/json",
        },
        credentials: "same-origin",
        body: JSON.stringify(json),
    });
    await assertResponse(response, redirectOnUnauthorized);

    const contentType = response.headers.get("Content-Type");
    if (contentType && contentType.includes("application/json")) {
        log.debug("Parsing JSON response from POST");
        const respJson = await response.json();
        return respJson as T;
    }

    log.debug("Unknown or no content type in response from POST, returning body text");
    const respText = await response.text();
    return respText as T;
}
