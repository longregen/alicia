/**
 * Authenticated fetch utility.
 * Wraps the native fetch to automatically include the X-User-ID header
 * for backend authentication.
 */
import { getDeviceId } from './deviceId';

/**
 * Get the user ID for API authentication.
 * Uses the persistent device ID to identify the user.
 */
export function getUserId(): string {
  return `user_${getDeviceId()}`;
}

/**
 * Authenticated fetch that automatically includes the X-User-ID header.
 * Use this instead of native fetch for all API calls that require authentication.
 */
export async function authFetch(
  url: string,
  options?: RequestInit
): Promise<Response> {
  const headers = new Headers(options?.headers);
  headers.set('X-User-ID', getUserId());

  return fetch(url, {
    ...options,
    headers,
  });
}
