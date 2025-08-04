import { readPropOrDefault, type JSONResponse } from "$lib/utils/json";
import { apiGet, type FetchFn } from "../../api-object";

export class RequestMetrics {
    readonly httpProxyRequests: number;
    readonly httpsProxyRequests: number;
    readonly bytesServed: number;

    constructor(json: JSONResponse) {
        this.httpProxyRequests = readPropOrDefault("http_proxy_requests", json, 0);
        this.httpsProxyRequests = readPropOrDefault("https_proxy_requests", json, 0);
        this.bytesServed = readPropOrDefault("bytes_served", json, 0);
    }
}

export async function getRequestMetrics(fetchFn: FetchFn = fetch): Promise<RequestMetrics> {
    return apiGet("/metrics/requests", RequestMetrics, fetchFn);
}
