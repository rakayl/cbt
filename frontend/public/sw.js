const DB_NAME = 'enterprise-cbt-offline';
const DB_VERSION = 1;

function openDatabase() {
  return new Promise((resolve, reject) => {
    const request = indexedDB.open(DB_NAME, DB_VERSION);
    request.onupgradeneeded = () => {
      const db = request.result;
      for (const store of ['answers', 'proctoringEvents', 'syncQueue']) {
        if (!db.objectStoreNames.contains(store)) db.createObjectStore(store, { keyPath: 'id' });
      }
    };
    request.onsuccess = () => resolve(request.result);
    request.onerror = () => reject(request.error);
  });
}

async function getQueued() {
  const db = await openDatabase();
  return new Promise((resolve, reject) => {
    const tx = db.transaction('syncQueue', 'readonly');
    const request = tx.objectStore('syncQueue').getAll();
    request.onsuccess = () => resolve(request.result || []);
    request.onerror = () => reject(request.error);
  });
}

async function removeQueued(id) {
  const db = await openDatabase();
  return new Promise((resolve, reject) => {
    const tx = db.transaction('syncQueue', 'readwrite');
    tx.objectStore('syncQueue').delete(id);
    tx.oncomplete = () => resolve();
    tx.onerror = () => reject(tx.error);
  });
}

self.addEventListener('install', (event) => event.waitUntil(self.skipWaiting()));
self.addEventListener('activate', (event) => event.waitUntil(self.clients.claim()));
self.addEventListener('sync', (event) => {
  if (event.tag === 'sync-answers') {
    event.waitUntil((async () => {
      const queued = await getQueued();
      for (const item of queued) {
        const response = await fetch('/api/v1' + item.endpoint, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json', ...(item.headers || {}) },
          credentials: 'include',
          body: JSON.stringify(item.payload),
        });
        if (!response.ok) throw new Error('background sync failed');
        await removeQueued(item.id);
      }
    })());
  }
});
