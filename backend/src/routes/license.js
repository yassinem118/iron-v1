import { Router } from 'express';
import { requireUser } from '../middleware/auth.js';
import { License } from '../models/license.js';
import { ActivationRequest } from '../models/activationRequest.js';
import { User } from '../models/user.js';
import { audit } from '../utils/audit.js';

export const licenseRouter = Router();
licenseRouter.use(requireUser);

async function ensureLicense(userId) {
  return License.findOneAndUpdate(
    { user: userId },
    { $setOnInsert: { user: userId, status: 'trial', trialRemaining: 2 } },
    { upsert: true, new: true, setDefaultsOnInsert: true }
  );
}

licenseRouter.get('/status', async (req, res, next) => {
  try {
    const license = await ensureLicense(req.auth.sub);
    if (license.status === 'active' && license.expiresAt && license.expiresAt <= new Date()) {
      license.status = 'expired'; await license.save();
    }
    res.json({
      status: license.status,
      trialRemaining: license.trialRemaining,
      startsAt: license.startsAt,
      expiresAt: license.expiresAt,
      renewalRequestedAt: license.renewalRequestedAt,
    });
  } catch (e) { next(e); }
});

// This endpoint counts an attempt; it does not claim to predict an unrevealed outcome.
licenseRouter.post('/trial/consume', async (req, res, next) => {
  try {
    const license = await ensureLicense(req.auth.sub);
    if (license.status === 'active' && license.expiresAt > new Date()) {
      return res.json({ allowed: true, mode: 'licensed', trialRemaining: license.trialRemaining });
    }
    if (license.status === 'active' && license.expiresAt <= new Date()) {
      license.status = 'expired';
    }
    if (license.status !== 'trial' || license.trialRemaining <= 0) {
      return res.status(402).json({ allowed: false, status: license.status, trialRemaining: 0, message: 'Trial exhausted or license inactive.' });
    }
    license.trialRemaining -= 1;
    if (license.trialRemaining === 0) license.status = 'pending';
    await license.save();
    audit('license.trial_consumed', { userId: String(req.auth.sub), remaining: license.trialRemaining });
    res.json({ allowed: true, mode: 'trial', trialRemaining: license.trialRemaining, status: license.status });
  } catch (e) { next(e); }
});

licenseRouter.post('/activation/request', async (req, res, next) => {
  try {
    const user = await User.findById(req.auth.sub).select('accountId');
    if (!user) return res.status(401).json({ error: 'Session expired' });
    const existing = await ActivationRequest.findOne({ user: user._id, status: 'pending' });
    if (existing) return res.status(200).json({ requestId: existing.id, status: existing.status });
    const request = await ActivationRequest.create({ user: user._id, accountId: user.accountId });
    const license = await ensureLicense(user._id);
    license.status = 'pending'; license.renewalRequestedAt = new Date(); await license.save();
    audit('license.activation_requested', { userId: String(user._id), requestId: request.id });
    res.status(201).json({ requestId: request.id, status: request.status });
  } catch (e) { next(e); }
});
