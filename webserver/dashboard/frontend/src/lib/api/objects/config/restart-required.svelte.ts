import { apiGet, type FetchFn } from "$lib/api/api-methods";

export class RestartRequiredResponse {
    constructor(json: Record<string, unknown>) {
        Object.assign(this, json);
    }

    readonly restart_required: boolean = false;
}

export async function getRestartRequired(
    fetchFn: FetchFn = fetch,
): Promise<RestartRequiredResponse> {
    return apiGet("/config/restart-required", RestartRequiredResponse, fetchFn);
}
