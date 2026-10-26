import mongoose from 'mongoose';
import { app } from './app.js';
import { env } from './config/env.js';

async function main() {
  try {
    // 1. الاتصال بقاعدة البيانات أولاً
    await mongoose.connect(env.mongoUri);
    console.log('Connected to MongoDB successfully');

    // 2. تشغيل السيرفر باش يبقى ديما شغال (Long-running)
    const server = app.listen(env.port, () => {
      console.log(`Iron v1 API listening on port ${env.port}`);
    });

    // 3. التعامل مع الإغلاق الآمن
    function shutdown(signal) {
      console.log(`${signal}: shutting down gracefully`);
      server.close(async () => {
        await mongoose.disconnect();
        console.log('MongoDB disconnected');
        process.exit(0);
      });
      setTimeout(() => {
        console.error('Forced shutdown due to timeout');
        process.exit(1);
      }, 10000).unref();
    }

    process.on('SIGTERM', () => shutdown('SIGTERM'));
    process.on('SIGINT', () => shutdown('SIGINT'));

  } catch (error) {
    console.error('Failed to start the server:', error);
    process.exit(1);
  }
}

main();