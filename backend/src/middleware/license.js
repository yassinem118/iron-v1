import { License } from '../models/license.js';

export async function requireActiveLicense(req, res, next) {
  try {
    const license = await License.findOne({ user: req.auth.sub });
    if (!license) return res.status(402).json({ error: 'trial_not_initialized', message: 'License record is not initialized.' });
    if (license.status !== 'active' || !license.expiresAt || license.expiresAt <= new Date()) {
      if (license.status === 'active' && license.expiresAt <= new Date()) {
        license.status = 'expired';
        await license.save();
      }
      return res.status(402).json({ error: 'license_inactive', status: license.status, message: 'An active license is required.' });
    }
    req.license = license;
    return next();
  } catch (err) { return next(err); }
}
