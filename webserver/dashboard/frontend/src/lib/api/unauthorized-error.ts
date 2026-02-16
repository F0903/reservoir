export default class UnauthorizedError extends Error {
    constructor() {
        super("Unauthorized, redirecting to login.");
        this.name = "UnauthorizedError";
    }
}
