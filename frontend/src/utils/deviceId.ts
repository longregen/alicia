/**
 * Device ID utility for persistent cross-device identification
 * Stores device ID in localStorage to maintain consistency across page loads
 */

const DEVICE_ID_KEY = 'alicia_device_id';

/**
 * Generate a random device ID
 */
function generateDeviceId(): string {
  const timestamp = Date.now();
  const random = Math.random().toString(36).substring(2, 15);
  return `device_${timestamp}_${random}`;
}

/**
 * Get or create persistent device ID from localStorage
 */
export function getDeviceId(): string {
  try {
    // Try to get existing device ID from localStorage
    let deviceId = localStorage.getItem(DEVICE_ID_KEY);

    if (!deviceId) {
      // Generate new device ID if none exists
      deviceId = generateDeviceId();
      localStorage.setItem(DEVICE_ID_KEY, deviceId);
    }

    return deviceId;
  } catch (error) {
    // Fallback if localStorage is not available (e.g., private browsing)
    console.warn('localStorage not available, using temporary device ID:', error);
    return generateDeviceId();
  }
}

/**
 * Reset device ID (useful for testing or logout)
 */
export function resetDeviceId(): void {
  try {
    localStorage.removeItem(DEVICE_ID_KEY);
  } catch (error) {
    console.warn('Failed to reset device ID:', error);
  }
}
