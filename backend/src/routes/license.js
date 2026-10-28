import { Router } from 'express';
import argon2 from 'argon2';
import { requireUser } from '../middleware/auth.js';
import { License } from '../models/license.js';
import { ActivationRequest } from '../models/activationRequest.js';
import { User } from '../models/user.js';
import { audit } from '../utils/audit.js';

export const licenseRouter = Router();
licenseRouter.use(requireUser);

async function ensureLicense(userId) {
  try {
    return await License.findOneAndUpdate(
      { user: userId },
      { $setOnInsert: { user: userId, status: 'trial', trialRemaining: 2 } },
      { upsert: true, new: true, setDefaultsOnInsert: true }
    );
  } catch (error) {
    // A concurrent first request can race on the unique user index.
    if (error?.code !== 11000) throw error;
    return License.findOne({ user: userId });
  }
}

async function expireIfNeeded(license) {
  if (license.status === 'active' && license.expiresAt && license.expiresAt <= new Date()) {
    license.status = 'expired';
    await license.save();
  }
  return license;
}

licenseRouter.get('/status', async (req, res, next) => {
  try {
    const license = await expireIfNeeded(await ensureLicense(req.auth.sub));
    return res.json({ status: license.status, trialRemaining: license.trialRemaining, startsAt: license.startsAt, expiresAt: license.expiresAt, renewalRequestedAt: license.renewalRequestedAt });
  } catch (error) { return next(error); }
});

// Atomically consume one of the two server-tracked trial uses.
licenseRouter.post('/trial/consume', async (req, res, next) => {
  try {
    let license = await expireIfNeeded(await ensureLicense(req.auth.sub));
    if (license.status === 'active' && license.expiresAt > new Date()) {
      return res.json({ allowed: true, mode: 'licensed', trialRemaining: license.trialRemaining });
    }
    if (license.status !== 'trial' || license.trialRemaining <= 0) {
      return res.status(402).json({ allowed: false, status: license.status, trialRemaining: 0, message: 'Trial exhausted or license inactive.' });
    }
    const consumed = await License.findOneAndUpdate(
      { _id: license._id, status: 'trial', trialRemaining: { $gt: 0 } },
      { $inc: { trialRemaining: -1 } },
      { new: true }
    );
    if (!consumed) {
      license = await License.findById(license._id);
      return res.status(402).json({ allowed: false, status: license?.status || 'pending', trialRemaining: license?.trialRemaining || 0, message: 'Trial exhausted or license inactive.' });
    }
    if (consumed.trialRemaining === 0) {
      consumed.status = 'pending';
      await consumed.save();
    }
    audit('license.trial_consumed', { userId: String(req.auth.sub), remaining: consumed.trialRemaining });
    return res.json({ allowed: true, mode: 'trial', trialRemaining: consumed.trialRemaining, status: consumed.status });
  } catch (error) { return next(error); }
});

// Redeem a customer-specific activation code. The raw code is never stored.
licenseRouter.post('/activate', async (req, res, next) => {
  try {
    const code = typeof req.body?.code === 'string' ? req.body.code.trim() : '';
    if (!/^[A-Z0-9-]{12,64}$/.test(code)) return res.status(400).json({ error: 'Invalid activation code' });
    const license = await License.findOne({ user: req.auth.sub }).select('+activationCodeHash');
    if (!license || !license.activationCodeHash || !['pending', 'trial', 'expired'].includes(license.status)) {
      return res.status(400).json({ error: 'No redeemable activation code is available' });
    }
    const valid = await argon2.verify(license.activationCodeHash, code);
    if (!valid) {
      audit('license.activation_failed', { userId: String(req.auth.sub) });
      return res.status(400).json({ error: 'Invalid activation code' });
    }
    const now = new Date();
    const activated = await License.findOneAndUpdate(
      { _id: license._id, activationCodeHash: license.activationCodeHash, status: { $in: ['pending', 'trial', 'expired'] } },
      { $set: { status: 'active', startsAt: now, activatedAt: now, expiresAt: new Date(now.getTime() + license.licenseDurationDays * 86400000) }, $unset: { activationCodeHash: 1 } },
      { new: true }
    );
    if (!activated) return res.status(409).json({ error: 'Activation code has already been used' });
    audit('license.activated', { userId: String(req.auth.sub), expiresAt: activated.expiresAt });
    return res.json({ success: true, status: activated.status, expiresAt: activated.expiresAt });
  } catch (error) { return next(error); }
});

licenseRouter.post('/activation/request', async (req, res, next) => {
  try {
    const user = await User.findById(req.auth.sub).select('accountId');
    if (!user) return res.status(401).json({ error: 'Session expired' });
    const existing = await ActivationRequest.findOne({ user: user._id, status: 'pending' });
    if (existing) return res.status(200).json({ requestId: existing.id, status: existing.status });
    const request = await ActivationRequest.create({ user: user._id, accountId: user.accountId });
    const license = await ensureLicense(user._id);
    if (license.status !== 'active') { license.status = 'pending'; license.renewalRequestedAt = new Date(); await license.save(); }
    audit('license.activation_requested', { userId: String(user._id), requestId: request.id });
    return res.status(201).json({ requestId: request.id, status: request.status });
  } catch (error) { return next(error); }
});
