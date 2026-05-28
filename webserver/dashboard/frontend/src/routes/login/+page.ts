import { browser } from "$app/environment";
import { redirect } from "@sveltejs/kit";
import type { PageLoad } from "./$types";

type BootstrapStatus = {
    bootstrap_required?: boolean;
};

export const load: PageLoad = async ({ fetch, url }) => {
    if (browser && (await bootstrapRequired(fetch))) {
        throw redirect(302, "/bootstrap");
    }

    return {
        return: url.searchParams.get("return"),
    };
};

async function bootstrapRequired(fetchFn: typeof fetch): Promise<boolean> {
    try {
        const response = await fetchFn("/api/auth/bootstrap", {
            credentials: "same-origin",
        });
        if (!response.ok) {
            return false;
        }

        const status = (await response.json()) as BootstrapStatus;
        return status.bootstrap_required === true;
    } catch {
        return false;
    }
}
