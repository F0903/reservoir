import { apiGet, type APIObjectConstructor, type FetchFn } from "$lib/api/api-object";
import { setPropIfChanged } from "$lib/utils/objects";

export class RequestMetrics {
    httpProxyRequests: number = $state(0);
    httpsProxyRequests: number = $state(0);
    bytesServed: number = $state(0);

    constructor(json: Record<string, unknown>) {
        this.updateFrom(json);
    }

    updateFrom = (json: Record<string, unknown>) => {
        setPropIfChanged(
            "http_proxy_requests",
            json,
            this.httpProxyRequests,
            (value) => (this.httpProxyRequests = value as number),
        );
        setPropIfChanged(
            "https_proxy_requests",
            json,
            this.httpsProxyRequests,
            (value) => (this.httpsProxyRequests = value as number),
        );
        setPropIfChanged(
            "bytes_served",
            json,
            this.bytesServed,
            (value) => (this.bytesServed = value as number),
        );
    };
}

export async function getRequestMetrics<C extends APIObjectConstructor<T>, T>(
    type: C = RequestMetrics as C,
    fetchFn: FetchFn = fetch,
): Promise<T> {
    return apiGet<T>("/metrics/requests", type, fetchFn);
}
