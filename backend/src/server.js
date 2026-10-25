import mongoose from 'mongoose';
import { app } from './app.js';
import { env } from './config/env.js';

await mongoose.connect(env.mongoUri);
const server = app.listen(env.port, () => console.log(`Iron v1 API listening on ${env.port}`));
async function shutdown(signal) {
  console.log(`${signal}: shutting down`);
  server.close(async () => { await mongoose.disconnect(); process.exit(0); });
  setTimeout(() => process.exit(1), 10000).unref();
}
process.on('SIGTERM', () => shutdown('SIGTERM'));
process.on('SIGINT', () => shutdown('SIGINT'));
