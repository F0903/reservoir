import { apiGet, type FetchFn } from "$lib/api/api-helpers";

export type Version = {
    version: string;
};

export async function version(fetchFn: FetchFn = fetch): Promise<Readonly<Version>> {
    return apiGet("/version", fetchFn);
}
