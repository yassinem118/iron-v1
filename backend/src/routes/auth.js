import { Router } from 'express';
import argon2 from 'argon2';
import jwt from 'jsonwebtoken';
import { z } from 'zod';
import { User } from '../models/user.js';
import { env } from '../config/env.js';
import { requireUser } from '../middleware/auth.js';
import { audit } from '../utils/audit.js';

export const authRouter = Router();
const accountIdSchema = z.string().trim().min(3).max(64).regex(/^[a-zA-Z0-9_.-]+$/);
const registerSchema = z.object({ accountId: accountIdSchema, password: z.string().min(12).max(128) });
const loginSchema = z.object({ accountId: accountIdSchema, password: z.string().min(1).max(128) });

function issueSession(user, res) {
  const token = jwt.sign({ sub: user.id }, env.jwtSecret, {
    expiresIn: '30m', issuer: 'iron-v1', audience: 'iron-v1-client',
  });
  res.cookie('iron_session', token, {
    httpOnly: true, secure: env.production, sameSite: 'strict', maxAge: 30 * 60 * 1000,
  });
}

authRouter.post('/register', async (req, res, next) => {
  try {
    const parsed = registerSchema.safeParse(req.body);
    if (!parsed.success) return res.status(400).json({ error: 'Invalid account ID or password (password must be 12+ characters)' });
    const passwordHash = await argon2.hash(parsed.data.password, { type: argon2.argon2id });
    const user = await User.create({ accountId: parsed.data.accountId, passwordHash });
    audit('account.registered', { accountId: user.accountId });
    issueSession(user, res);
    return res.status(201).json({ accountId: user.accountId, created: true });
  } catch (error) {
    if (error?.code === 11000) return res.status(409).json({ error: 'Account ID already exists' });
    return next(error);
  }
});

authRouter.post('/login', async (req, res, next) => {
  try {
    const parsed = loginSchema.safeParse(req.body);
    if (!parsed.success) return res.status(400).json({ error: 'Invalid credentials' });
    const user = await User.findOne({ accountId: parsed.data.accountId }).select('+passwordHash');
    if (!user || !user.active || !(await argon2.verify(user.passwordHash, parsed.data.password))) {
      audit('account.login_failed', { accountId: parsed.data.accountId });
      return res.status(401).json({ error: 'Invalid account ID or password' });
    }
    user.lastLoginAt = new Date();
    await user.save();
    issueSession(user, res);
    audit('account.login_success', { accountId: user.accountId });
    return res.json({ accountId: user.accountId, expiresIn: 1800 });
  } catch (error) { return next(error); }
});

// First attempt registers a new Iron account; existing accounts must authenticate.
authRouter.post('/connect', async (req, res, next) => {
  try {
    const parsed = registerSchema.safeParse(req.body);
    if (!parsed.success) return res.status(400).json({ success: false, message: 'Invalid Iron account ID or password', redirect: false });
    const { accountId, password } = parsed.data;
    let user = await User.findOne({ accountId }).select('+passwordHash');
    let created = false;
    if (!user) {
      const passwordHash = await argon2.hash(password, { type: argon2.argon2id });
      try {
        user = await User.create({ accountId, passwordHash });
        created = true;
        audit('account.registered', { accountId });
      } catch (error) {
        if (error?.code !== 11000) throw error;
        user = await User.findOne({ accountId }).select('+passwordHash');
      }
    }
    if (!user || !user.active || !(await argon2.verify(user.passwordHash, password))) {
      audit('account.connect_failed', { accountId });
      return res.status(401).json({ success: false, message: 'Invalid Iron account ID or password', redirect: false });
    }
    user.lastLoginAt = new Date();
    await user.save();
    issueSession(user, res);
    audit('account.connect_success', { accountId: user.accountId, created });
    return res.status(200).json({ success: true, message: 'Connected successfully', redirect: true, accountId: user.accountId });
  } catch (error) { return next(error); }
});

authRouter.post('/logout', (_req, res) => res.clearCookie('iron_session', { httpOnly: true, secure: env.production, sameSite: 'strict' }).status(204).end());
authRouter.get('/me', requireUser, async (req, res, next) => {
  try {
    const user = await User.findById(req.auth.sub).select('accountId active createdAt lastLoginAt');
    if (!user) return res.status(401).json({ error: 'Session expired' });
    return res.json(user);
  } catch (error) { return next(error); }
});
