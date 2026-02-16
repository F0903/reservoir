import { apiGetTextStream, type FetchFn } from "../../api-helpers";

export function getLogStream(fetchFn: FetchFn = fetch): Promise<ReadableStream<string>> {
    return apiGetTextStream("/log", fetchFn);
}
