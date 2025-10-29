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
export const DefaultRedirectOptions: LoginRedirectOptions = { returnToLastWindow: true };

async function redirectToLogin(redirect: LoginRedirectOptions): Promise<void> {
    const params = new URLSearchParams();

    const location = window.location.pathname;
    const isOnLoginPage = location === "/login";

    if (redirect.returnToLastWindow) {
        const windowParams = new URLSearchParams(window.location.search);
        // If the user is on the login page, we don't want to redirect back to the login page again.
        // Instead, we want to redirect to the page they were trying to access before they were redirected to the login page.
        const returnTo = isOnLoginPage
            ? windowParams.get("return")
            : window.location.pathname + window.location.search;
        if (returnTo) {
            log.debug("Redirecting to login from API call, will return to:", returnTo);
            params.append("return", returnTo);
        }
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
    redirectOnUnauthorized: LoginRedirectOptions | null = DefaultRedirectOptions,
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
    redirectOnUnauthorized: LoginRedirectOptions | null = DefaultRedirectOptions,
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
    fetchFn: FetchFn = fetch,
    redirectOnUnauthorized: LoginRedirectOptions | null = DefaultRedirectOptions,
): Promise<T> {
    const resp = await getAssert(endpoint, fetchFn, redirectOnUnauthorized);
    const json = await resp.json();
    return json as T;
}

export async function apiPatch<T>(
    endpoint: string,
    json: Record<string, unknown>,
    fetchFn: FetchFn = fetch,
    redirectOnUnauthorized: LoginRedirectOptions | null = DefaultRedirectOptions,
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
    redirectOnUnauthorized: LoginRedirectOptions | null = DefaultRedirectOptions,
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
