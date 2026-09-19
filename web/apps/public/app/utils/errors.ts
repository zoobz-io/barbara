import { ConflictError, HttpError } from "@barbara/api-sdk";

/**
 * A user-facing message for a failed name-carrying API call: conflicts name
 * the collision, other HTTP errors speak for themselves, anything else gets
 * a generic retry line.
 */
export function apiErrorMessage(err: unknown, name: string): string {
  if (err instanceof ConflictError) {
    return `"${name}" already exists.`;
  }
  if (err instanceof HttpError) {
    return err.message;
  }
  return "Something went wrong — try again.";
}
