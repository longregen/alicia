const DEVICE_ID_KEY = 'alicia_device_id';
const USER_ID_KEY = 'alicia_user_id';
const MAX_USER_ID_LENGTH = 256;

function generateDeviceId(): string {
  return `device_${crypto.randomUUID()}`;
}

export function getDeviceId(): string {
  try {
    let deviceId = localStorage.getItem(DEVICE_ID_KEY);
    if (!deviceId) {
      deviceId = generateDeviceId();
      localStorage.setItem(DEVICE_ID_KEY, deviceId);
    }
    return deviceId;
  } catch {
    return generateDeviceId();
  }
}

export function getUserId(): string {
  try {
    const customUserId = localStorage.getItem(USER_ID_KEY);
    if (customUserId) {
      return customUserId;
    }
  } catch {
    // localStorage unavailable, fall through to device ID
  }
  return `user_${getDeviceId()}`;
}

export function setUserId(userId: string | null): void {
  try {
    if (userId && userId.length <= MAX_USER_ID_LENGTH) {
      localStorage.setItem(USER_ID_KEY, userId);
    } else {
      localStorage.removeItem(USER_ID_KEY);
    }
  } catch {
    // localStorage unavailable
  }
}

export function getCustomUserId(): string | null {
  try {
    return localStorage.getItem(USER_ID_KEY);
  } catch {
    return null;
  }
}
