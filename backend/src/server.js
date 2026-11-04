import mongoose from 'mongoose';
import { app } from './app.js';
import { env } from './config/env.js';
import cors from 'cors';
import express from 'express';

// تفعيل إمكانية قراءة JSON والـ CORS إذا لم تكن مفعلة مسبقاً في app.js
app.use(express.json({ limit: '64kb' }));
app.use(cors());

// تخزين مؤقت معزول في الذاكرة لكل زبون (Session Isolation) لتفادي تضخيم قاعدة البيانات
const userStores = new Map();

// 1. نقطة النهاية لاستقبال الإطارات والبيانات من سكريبت الفريدا
app.post('/api/ingest', (req, res) => {
    const { userId, source, kind, payload } = req.body;

    if (!userId || !payload) {
        return res.status(400).json({ error: "userId and payload are required" });
    }

    // حفظ أو تحديث حالة اللعبة الخاصة بهذا المستخدم فقط في الذاكرة
    userStores.set(userId, {
        payload,
        source: source || "unknown",
        kind: kind || "telemetry",
        updatedAt: new Date()
    });

    return res.status(202).json({ accepted: true, message: "Frame isolated and updated for user" });
});

// 2. نقطة النهاية الخاصة بالـ Dashboard لجلب حالة اللعبة الخاصة بالزبون لحظياً
app.get('/api/dashboard/state', (req, res) => {
    const userId = parseInt(req.query.userId);
    if (!userId || isNaN(userId)) {
        return res.status(400).json({ error: "Valid userId query parameter is required" });
    }

    const sessionData = userStores.get(userId);
    if (!sessionData) {
        return res.status(404).json({ error: "No active game telemetry found for this user session" });
    }

    return res.json({
        success: true,
        userId: userId,
        data: sessionData
    });
});

async function main() {
  try {
    // الاتصال بقاعدة البيانات أولاً مع إضافة خيارات المهلة الزمنية لتفادي التوقف في DomCloud
    await mongoose.connect(env.mongoUri, {
      serverSelectionTimeoutMS: 5000, // مهلة 5 ثواني كحد أقصى للبحث عن السيرفر
      socketTimeoutMS: 45000,         // مهلة الاتصال بالـ Socket
    });
    console.log('Connected to MongoDB successfully');

    // تشغيل السيرفر باش يبقى ديما شغال (Long-running)
    const server = app.listen(env.port, () => {
      console.log(`Iron v1 API listening on port ${env.port}`);
    });

    // التعامل مع الإغلاق الآمن
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