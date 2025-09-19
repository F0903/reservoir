export function snakeCaseToCamelCase(s: string): string {
    return s.replace(/_+([a-z0-9])/gi, (_m, c: string) => c.toUpperCase());
}
