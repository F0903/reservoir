export type LoadableState = {
    tag: "loading" | "ok" | "error";
    errorMsg: string | null;
};

export interface Loadable {
    getLoadableState(): LoadableState;
}
