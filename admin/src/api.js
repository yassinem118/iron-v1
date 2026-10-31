const API = (import.meta.env.VITE_API_URL || 'http://localhost:3000').replace(/\/$/, '');
async function request(path, key, options = {}) {
  const r = await fetch(`${API}${path}`, { ...options, headers: {
    ...(key ? { 'x-admin-key': key } : {}),
    ...(options.body ? { 'content-type': 'application/json' } : {}),
    ...options.headers,
  }});
  if (!r.ok) { let m = `API request failed (${r.status})`; try { const b = await r.json(); m = b.error || b.message || m; } catch {} throw new Error(m); }
  if (r.status === 204) return null;
  return r.json();
}
export const listUsers = k => request('/api/admin/users?limit=100', k);
export const createUser = (k, d) => request('/api/admin/users', k, { method:'POST', body:JSON.stringify(d) });
export const updateUser = (k, id, d) => request(`/api/admin/users/${encodeURIComponent(id)}`, k, { method:'PATCH', body:JSON.stringify(d) });
export const setActive = (k, id, active) => request(`/api/admin/users/${encodeURIComponent(id)}/status`, k, { method:'PATCH', body:JSON.stringify({active}) });
export const listLicenses = k => request('/api/admin/licenses', k);
export const issueActivationCode = (k, id, days) => request(`/api/admin/licenses/users/${encodeURIComponent(id)}/code`, k, { method:'POST', body:JSON.stringify({days:Number(days)}) });
export const listActivationRequests = k => request('/api/admin/licenses/requests', k);
export const reviewActivationRequest = (k, id, action, days) => request(`/api/admin/licenses/requests/${encodeURIComponent(id)}/${action}`, k, { method:'POST', body:JSON.stringify({days:Number(days)}) });
