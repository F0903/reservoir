# reservoir Dashboard

This is the frontend for reservoir, a web dashboard for configuring and monitoring the proxy.

## Development

Before starting the frontend, make sure you have [**Node** installed](https://nodejs.org/en/download) and **pnpm** enabled *(run the command `corepack enable pnpm`)*

Then install the frontend dependencies by running `pnpm install` in the `frontend/` directory.

The frontend is designed to be embedded in- and served by the proxy, therefore you **must** have the proxy running locally as well for the dashboard to pull data from, if you want to run it in development mode.

### Starting manually

To start the frontend in development mode, run the following command:

```sh
pnpm run dev
```
