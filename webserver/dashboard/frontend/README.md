# reservoir Dashboard

This is the frontend for reservoir, which is a web interface for managing and monitoring the proxy.

## Development

Before starting the frontend, make sure you have [**Node** installed](https://nodejs.org/en/download) and **pnpm** enabled *(run the command `corepack enable pnpm`)*

Then install the frontend dependencies by running `pnpm install` in the `frontend/` directory.

### VS Code Debugging (Recommended)

The easiest and recommended way to start the frontend in development mode is to use the provided VS Code launch configuration.

To use it, follow these steps:

1. Open the project in VS Code
2. Open the Debug panel (Ctrl+Shift+D)
3. Select either "Proxy + Dashboard" (recommended) or "Dashboard" (you will need to start the proxy separately)
4. Start debugging :)

### Starting manually

To start the frontend in development mode, run the following command:

```sh
pnpm run dev
```
