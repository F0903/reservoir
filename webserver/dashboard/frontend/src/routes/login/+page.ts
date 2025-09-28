import type { PageLoad } from "./$types";

export const load: PageLoad = ({ url }) => {
    return {
        return: url.searchParams.get("return"),
    };
};
