export function getDeviceFingerprint() {
  const storageKey = 'cbt.device.id';
  let deviceId = localStorage.getItem(storageKey);
  if (!deviceId) {
    deviceId = crypto.randomUUID();
    localStorage.setItem(storageKey, deviceId);
  }
  const raw = [
    deviceId,
    navigator.userAgent,
    navigator.language,
    screen.width,
    screen.height,
    screen.colorDepth,
    Intl.DateTimeFormat().resolvedOptions().timeZone,
  ].join('|');
  return btoa(unescape(encodeURIComponent(raw))).slice(0, 256);
}

export function getDeviceName() {
  const platform = navigator.userAgentData?.platform || navigator.platform || 'Browser';
  return `${platform} ${screen.width}x${screen.height}`;
}
