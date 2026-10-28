import { Router } from 'express';
import { randomBytes } from 'node:crypto';
import argon2 from 'argon2';
import mongoose from 'mongoose';
import { User } from '../models/user.js';
import { z } from 'zod';
import { requireAdmin } from '../middleware/auth.js';
import { License } from '../models/license.js';
import { ActivationRequest } from '../models/activationRequest.js';
import { audit } from '../utils/audit.js';

export const adminLicensesRouter = Router();
adminLicensesRouter.use(requireAdmin);

// View license status alongside the Iron account ID; never expose code hashes.
adminLicensesRouter.get('/', async (_req, res, next) => {
  try {
    const licenses = await License.find({})
      .populate('user', 'accountId active createdAt')
      .select('-activationCodeHash')
      .sort({ updatedAt: -1 }).limit(500).lean();
    return res.json({ licenses });
  } catch (error) { return next(error); }
});

// Issue a high-entropy, single-use activation code for one Iron account.
adminLicensesRouter.post('/users/:userId/code', async (req, res, next) => {
  try {
    if (!mongoose.isValidObjectId(req.params.userId)) return res.status(400).json({ error: 'Invalid user ID' });
    const parsed = z.object({ days: z.number().int().min(1).max(365).default(15) }).safeParse(req.body ?? {});
    if (!parsed.success) return res.status(400).json({ error: 'days must be an integer from 1 to 365' });
    const user = await User.findById(req.params.userId).select('_id accountId active');
    if (!user || !user.active) return res.status(404).json({ error: 'Active Iron account not found' });

    // The plaintext is returned once to the authenticated admin; only its Argon2id hash is stored.
    const code = randomBytes(18).toString('hex').toUpperCase().match(/.{1,6}/g).join('-');
    const activationCodeHash = await argon2.hash(code, { type: argon2.argon2id });
    const now = new Date();
    const license = await License.findOneAndUpdate(
      { user: user._id },
      { $set: { status: 'pending', activationCodeHash, licenseDurationDays: parsed.data.days, startsAt: null, expiresAt: null, activatedAt: null, updatedByAdminAt: now } },
      { new: true, upsert: true, setDefaultsOnInsert: true, runValidators: true }
    );
    audit('license.activation_code_issued', { userId: String(user._id), accountId: user.accountId, days: parsed.data.days });
    return res.status(201).json({ success: true, accountId: user.accountId, code, status: license.status, durationDays: license.licenseDurationDays, message: 'Copy this code now; it will not be shown again.' });
  } catch (error) { return next(error); }
});


adminLicensesRouter.get('/requests', async (_req, res, next) => {
  try {
    const requests = await ActivationRequest.find({ status: 'pending' })
      .sort({ requestedAt: 1 }).limit(200).lean();
    res.json({ requests });
  } catch (e) { next(e); }
});

// Admin verifies payment outside this API, then explicitly approves a 15-day license.
adminLicensesRouter.post('/requests/:id/approve', async (req, res, next) => {
  try {
    const parsed = z.object({ days: z.number().int().min(1).max(15).default(15) }).safeParse(req.body ?? {});
    if (!parsed.success) return res.status(400).json({ error: 'days must be an integer from 1 to 15' });
    const request = await ActivationRequest.findOneAndUpdate(
      { _id: req.params.id, status: 'pending' },
      { $set: { status: 'approved', reviewedAt: new Date() } },
      { new: true }
    );
    if (!request) return res.status(404).json({ error: 'Pending activation request not found' });
    const now = new Date();
    const license = await License.findOneAndUpdate(
      { user: request.user },
      { $set: { status: 'active', startsAt: now, expiresAt: new Date(now.getTime() + parsed.data.days * 86400000), updatedByAdminAt: now } },
      { new: true, upsert: true, setDefaultsOnInsert: true }
    );
    audit('license.approved', { requestId: request.id, userId: String(request.user), days: parsed.data.days });
    res.json({ requestId: request.id, license: { status: license.status, startsAt: license.startsAt, expiresAt: license.expiresAt } });
  } catch (e) { next(e); }
});

adminLicensesRouter.post('/requests/:id/reject', async (req, res, next) => {
  try {
    const request = await ActivationRequest.findOneAndUpdate(
      { _id: req.params.id, status: 'pending' },
      { $set: { status: 'rejected', reviewedAt: new Date() } }, { new: true }
    );
    if (!request) return res.status(404).json({ error: 'Pending activation request not found' });
    audit('license.rejected', { requestId: request.id, userId: String(request.user) });
    res.json({ requestId: request.id, status: request.status });
  } catch (e) { next(e); }
});
