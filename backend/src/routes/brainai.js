import { Router } from 'express';
import { z } from 'zod';
import { requireUser } from '../middleware/auth.js';
import { requireActiveLicense } from '../middleware/license.js';
import { calibrate, summarizeOutcomes } from '../core/brainai/calibrator.js';
import { Outcome } from '../models/outcome.js';

export const brainaiRouter = Router();
brainaiRouter.use(requireUser, requireActiveLicense);

const observation = z.object({
  // Only outcomes already revealed by a legitimate source; never submit hidden-state guesses.
  predictedProbability: z.number().min(0).max(1),
  outcome: z.enum(['success', 'failure']),
  eventType: z.string().trim().min(1).max(40).default('observed'),
  durationMs: z.number().int().min(0).max(24 * 60 * 60 * 1000).optional(),
});

brainaiRouter.post('/observations', async (req, res, next) => {
  try {
    const parsed = observation.safeParse(req.body);
    if (!parsed.success) return res.status(400).json({ error: 'Invalid observation', details: parsed.error.flatten() });
    const doc = await Outcome.create({ user: req.auth.sub, ...parsed.data });
    res.status(201).json({ id: doc.id, stored: true });
  } catch (e) { next(e); }
});

brainaiRouter.get('/summary', async (req, res, next) => {
  try {
    const rows = await Outcome.find({ user: req.auth.sub }).sort({ createdAt: -1 }).limit(500).lean();
    const result = summarizeOutcomes(rows);
    res.json({ ...result, note: 'Calibration describes historical observations; it does not predict hidden or random game outcomes.' });
  } catch (e) { next(e); }
});

brainaiRouter.post('/calibrate', async (req, res, next) => {
  try {
    const rows = await Outcome.find({ user: req.auth.sub }).sort({ createdAt: -1 }).limit(500).lean();
    res.json(calibrate(rows));
  } catch (e) { next(e); }
});
