import { Router } from 'express';
import argon2 from 'argon2';
import { User } from '../models/user.js';
import { requireAdmin } from '../middleware/auth.js';
import { z } from 'zod';
import { audit } from '../utils/audit.js';

export const adminRouter = Router();
adminRouter.use(requireAdmin);

const accountIdSchema = z.string().trim().min(3).max(64).regex(/^[a-zA-Z0-9_.-]+$/);
const createSchema = z.object({
  accountId: accountIdSchema,
  password: z.string().min(12).max(128),
});
const updateSchema = z.object({
  accountId: accountIdSchema.optional(),
  // Admins may set a new password; the current password is never retrievable.
  password: z.string().min(12).max(128).optional(),
}).refine(v => v.accountId !== undefined || v.password !== undefined, {
  message: 'Provide accountId and/or a new password',
});

adminRouter.get('/users', async (req, res, next) => {
  try {
    const page = Math.max(1, Math.min(10000, Number(req.query.page) || 1));
    const limit = Math.max(1, Math.min(100, Number(req.query.limit) || 25));
    const [users, total] = await Promise.all([
      User.find({}, 'accountId active createdAt lastLoginAt')
        .sort({ createdAt: -1 }).skip((page - 1) * limit).limit(limit).lean(),
      User.countDocuments(),
    ]);
    res.json({ users, total, page, limit });
  } catch (e) { next(e); }
});

adminRouter.post('/users', async (req, res, next) => {
  try {
    const parsed = createSchema.safeParse(req.body);
    if (!parsed.success) return res.status(400).json({ error: 'Invalid account ID or password (password must be 12+ characters)' });
    const passwordHash = await argon2.hash(parsed.data.password, { type: argon2.argon2id });
    const user = await User.create({ accountId: parsed.data.accountId, passwordHash });
    audit('admin.user_created', { userId: user.id, accountId: user.accountId });
    return res.status(201).json({
      _id: user.id, accountId: user.accountId, active: user.active,
      createdAt: user.createdAt, passwordSet: true,
    });
  } catch (e) {
    if (e?.code === 11000) return res.status(409).json({ error: 'Account ID already exists' });
    next(e);
  }
});

adminRouter.patch('/users/:id', async (req, res, next) => {
  try {
    const parsed = updateSchema.safeParse(req.body);
    if (!parsed.success) return res.status(400).json({ error: 'Provide a valid account ID and/or a new password of at least 12 characters' });

    const updates = {};
    if (parsed.data.accountId !== undefined) updates.accountId = parsed.data.accountId;
    if (parsed.data.password !== undefined) {
      updates.passwordHash = await argon2.hash(parsed.data.password, { type: argon2.argon2id });
    }

    const user = await User.findByIdAndUpdate(req.params.id, { $set: updates }, {
      new: true, runValidators: true,
    }).select('accountId active createdAt lastLoginAt');
    if (!user) return res.status(404).json({ error: 'Account not found' });

    audit('admin.user_updated', {
      userId: user.id,
      accountId: user.accountId,
      accountIdChanged: parsed.data.accountId !== undefined,
      passwordReset: parsed.data.password !== undefined,
    });
    return res.json({ ...user.toObject(), passwordReset: parsed.data.password !== undefined });
  } catch (e) {
    if (e?.code === 11000) return res.status(409).json({ error: 'Account ID already exists' });
    next(e);
  }
});

adminRouter.patch('/users/:id/status', async (req, res, next) => {
  try {
    const parsed = z.object({ active: z.boolean() }).safeParse(req.body);
    if (!parsed.success) return res.status(400).json({ error: 'active must be boolean' });
    const user = await User.findByIdAndUpdate(req.params.id, { active: parsed.data.active }, { new: true }).select('accountId active createdAt');
    if (!user) return res.status(404).json({ error: 'Account not found' });
    audit('admin.user_status_changed', { userId: user.id, active: user.active });
    res.json(user);
  } catch (e) { next(e); }
});
