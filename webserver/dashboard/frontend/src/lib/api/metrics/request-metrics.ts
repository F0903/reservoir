import { apiGet, type JSONResponse } from "../api-object";

export class RequestMetrics {
    readonly httpProxyRequests: number;
    readonly httpsProxyRequests: number;
    readonly bytesServed: number;

    constructor(json: JSONResponse) {
        this.httpProxyRequests = json["http_proxy_requests"] as number;
        this.httpsProxyRequests = json["https_proxy_requests"] as number;
        this.bytesServed = json["bytes_served"] as number;
    }
}

export async function getRequestMetrics(): Promise<RequestMetrics> {
    return apiGet("/metrics/requests", RequestMetrics);
}
