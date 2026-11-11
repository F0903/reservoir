import { apiGetResponse, type FetchFn } from "../../api-helpers";

// The log file can be hundreds of megabytes and the viewer keeps 2000 lines,
// so only the end of the file is fetched.
const logTailBytes = 256 * 1024;

export async function getLogTail(fetchFn: FetchFn = fetch): Promise<string> {
    const resp = await apiGetResponse("/log", { Range: `bytes=-${logTailBytes}` }, fetchFn);
    const text = await resp.text();

    // A tail that starts inside the file most likely starts inside a line: drop that piece.
    const startsInsideFile =
        resp.status === 206 && !resp.headers.get("Content-Range")?.startsWith("bytes 0-");
    return startsInsideFile ? text.slice(text.indexOf("\n") + 1) : text;
}
