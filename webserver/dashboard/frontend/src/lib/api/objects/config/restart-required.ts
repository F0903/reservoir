import { apiGet, type FetchFn } from "$lib/api/api-methods";

export type RestartRequiredResponse = {
    restart_required: boolean;
};

export async function getRestartRequired(
    fetchFn: FetchFn = fetch,
): Promise<Readonly<RestartRequiredResponse>> {
    return apiGet("/config/restart-required", fetchFn);
}
