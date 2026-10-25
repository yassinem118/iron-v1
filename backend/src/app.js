import express from 'express';
import helmet from 'helmet';
import cors from 'cors';
import cookieParser from 'cookie-parser';
import rateLimit from 'express-rate-limit';
import { env } from './config/env.js';
import { authRouter } from './routes/auth.js';
import { adminRouter } from './routes/admin.js';
import { licenseRouter } from './routes/license.js';
import { brainaiRouter } from './routes/brainai.js';
import { adminLicensesRouter } from './routes/adminLicenses.js';

export const app = express();
app.disable('x-powered-by');
app.use(helmet());
app.use(cors({ origin: env.corsOrigin, credentials: true }));
app.use(express.json({ limit: '16kb' }));
app.use(cookieParser());
app.use('/api/auth', rateLimit({ windowMs: 15 * 60 * 1000, limit: 30, standardHeaders: 'draft-7', legacyHeaders: false }), authRouter);
app.use('/api/license', rateLimit({ windowMs: 15 * 60 * 1000, limit: 30, standardHeaders: 'draft-7', legacyHeaders: false }), licenseRouter);
app.use('/api/brainai', rateLimit({ windowMs: 15 * 60 * 1000, limit: 60, standardHeaders: 'draft-7', legacyHeaders: false }), brainaiRouter);
app.use('/api/admin/licenses', rateLimit({ windowMs: 60 * 1000, limit: 60, standardHeaders: 'draft-7', legacyHeaders: false }), adminLicensesRouter);
app.use('/api/admin', rateLimit({ windowMs: 60 * 1000, limit: 120, standardHeaders: 'draft-7', legacyHeaders: false }), adminRouter);
app.get('/health', (_req, res) => res.json({ status: 'ok', service: 'iron-v1-api' }));
app.use((err, _req, res, _next) => {
  // Log server-side details, but never send stack traces or secrets to clients.
  console.error('request_failed', err?.name || 'Error');
  res.status(500).json({ error: 'Internal server error' });
});
