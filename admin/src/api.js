const API = import.meta.env.VITE_API_URL || 'http://78.154.103.8:16673';

async function request(path, adminKey, options = {}) {
  const r = await fetch(`${API}${path}`, {
    ...options,
    headers: {
      'x-admin-key': adminKey,
      ...(options.body ? { 'content-type': 'application/json' } : {}),
      ...options.headers,
    },
  });
  if (!r.ok) {
    let message = `API error ${r.status}`;
    try { message = (await r.json()).error || message; } catch {}
    throw new Error(message);
  }
  if (r.status === 204) return null;
  return r.json();
}

export function listUsers(adminKey) {
  return request('/api/admin/users?limit=100', adminKey);
}
export function createUser(adminKey, data) {
  return request('/api/admin/users', adminKey, { method: 'POST', body: JSON.stringify(data) });
}
export function updateUser(adminKey, id, data) {
  return request(`/api/admin/users/${encodeURIComponent(id)}`, adminKey, { method: 'PATCH', body: JSON.stringify(data) });
}
export function setActive(adminKey, id, active) {
  return request(`/api/admin/users/${encodeURIComponent(id)}/status`, adminKey, { method: 'PATCH', body: JSON.stringify({ active }) });
}
