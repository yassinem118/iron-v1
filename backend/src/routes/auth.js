import { Router } from 'express';
import argon2 from 'argon2';
import jwt from 'jsonwebtoken';
import { z } from 'zod';
import { User } from '../models/user.js';
import { env } from '../config/env.js';
import { requireUser } from '../middleware/auth.js';
import { audit } from '../utils/audit.js';

export const authRouter = Router();
const credentials = z.object({
  accountId: z.string().trim().min(3).max(64).regex(/^[a-zA-Z0-9_.-]+$/),
  password: z.string().min(12).max(128),
});

// Register an Iron v1 account. Do not accept Melbet credentials here.
authRouter.post('/register', async (req, res, next) => {
  try {
    const parsed = credentials.safeParse(req.body);
    if (!parsed.success) return res.status(400).json({ error: 'Invalid account ID or password (password must be 12+ characters)' });
    const { accountId, password } = parsed.data;
    const passwordHash = await argon2.hash(password, { type: argon2.argon2id });
    await User.create({ accountId, passwordHash });
    audit('account.registered', { accountId });
    return res.status(201).json({ accountId, created: true });
  } catch (e) {
    if (e?.code === 11000) return res.status(409).json({ error: 'Account ID already exists' });
    next(e);
  }
});

authRouter.post('/login', async (req, res, next) => {
  try {
    const parsed = z.object({ accountId: z.string().trim().min(3).max(64), password: z.string().min(1).max(128) }).safeParse(req.body);
    if (!parsed.success) return res.status(400).json({ error: 'Invalid credentials' });
    const user = await User.findOne({ accountId: parsed.data.accountId }).select('+passwordHash');
    // Generic response prevents account enumeration.
    if (!user || !user.active || !(await argon2.verify(user.passwordHash, parsed.data.password))) {
      audit('account.login_failed', { accountId: parsed.data.accountId });
      return res.status(401).json({ error: 'Invalid account ID or password' });
    }
    user.lastLoginAt = new Date(); await user.save();
    const token = jwt.sign({ sub: user.id }, env.jwtSecret, { expiresIn: '30m', issuer: 'iron-v1', audience: 'iron-v1-client' });
    res.cookie('iron_session', token, { httpOnly: true, secure: env.production, sameSite: 'strict', maxAge: 30 * 60 * 1000 });
    audit('account.login_success', { accountId: user.accountId });
    return res.json({ accountId: user.accountId, expiresIn: 1800 });
  } catch (e) { next(e); }
});

authRouter.post('/logout', (_req, res) => res.clearCookie('iron_session', { httpOnly: true, secure: env.production, sameSite: 'strict' }).status(204).end());
authRouter.get('/me', requireUser, async (req, res, next) => {
  try { const user = await User.findById(req.auth.sub).select('accountId active createdAt lastLoginAt'); if (!user) return res.status(401).json({ error: 'Session expired' }); return res.json(user); }
  catch (e) { next(e); }
});
