import { Router } from 'express';
import { z } from 'zod';
import { requireAdmin } from '../middleware/auth.js';
import { License } from '../models/license.js';
import { ActivationRequest } from '../models/activationRequest.js';
import { audit } from '../utils/audit.js';

export const adminLicensesRouter = Router();
adminLicensesRouter.use(requireAdmin);

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
