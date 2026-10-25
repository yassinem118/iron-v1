import 'dotenv/config';

// Fail fast instead of booting with insecure production defaults.
for (const key of ['MONGODB_URI', 'JWT_SECRET', 'ADMIN_API_KEY']) {
  if (!process.env[key]) throw new Error(`Missing required environment variable: ${key}`);
}
if (process.env.NODE_ENV === 'production' && process.env.JWT_SECRET.length < 32) {
  throw new Error('JWT_SECRET must be at least 32 characters in production');
}
export const env = {
  port: Number(process.env.PORT || 8080),
  mongoUri: process.env.MONGODB_URI,
  jwtSecret: process.env.JWT_SECRET,
  adminApiKey: process.env.ADMIN_API_KEY,
  corsOrigin: process.env.CORS_ORIGIN || 'http://localhost:5173',
  production: process.env.NODE_ENV === 'production',
};
