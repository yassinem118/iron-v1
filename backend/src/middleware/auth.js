import jwt from 'jsonwebtoken';
import { env } from '../config/env.js';

export function requireUser(req, res, next) {
  const token = req.cookies?.iron_session || req.get('authorization')?.replace(/^Bearer\s+/i, '');
  if (!token) return res.status(401).json({ error: 'Authentication required' });
  try {
    req.auth = jwt.verify(token, env.jwtSecret, { issuer: 'iron-v1', audience: 'iron-v1-client' });
    return next();
  } catch { return res.status(401).json({ error: 'Invalid or expired session' }); }
}

// Admin access uses a separate secret; in production prefer SSO/OIDC + role-based access.
export function requireAdmin(req, res, next) {
  const supplied = req.get('x-admin-key') || '';
  const expected = env.adminApiKey;
  // Timing-safe compare without leaking secret length via a direct string comparison.
  const a = Buffer.from(supplied); const b = Buffer.from(expected);
  if (a.length !== b.length || !a.length) return res.status(403).json({ error: 'Forbidden' });
  let diff = 0; for (let i = 0; i < a.length; i++) diff |= a[i] ^ b[i];
  if (diff !== 0) return res.status(403).json({ error: 'Forbidden' });
  return next();
}
