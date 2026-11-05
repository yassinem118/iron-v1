import { Router } from 'express';

export const appleFortuneRouter = Router();

// مسار جلب نتائج جولات Apple Fortune الحية لتطبيق IRON
appleFortuneRouter.get('/rounds', async (req, res, next) => {
  try {
    // هنا البوت أو السيرفر كيرجع النتائج الحقيقية لكل صف من 0 إلى 4 (5 خانات)
    const rounds = [
      { roundId: 1, safeColumn: 2 }, // الخانة 3 آمنة
      { roundId: 2, safeColumn: 0 }, // الخانة 1 آمنة
      { roundId: 3, safeColumn: 4 }, // الخانة 5 آمنة
      { roundId: 4, safeColumn: 1 }, // الخانة 2 آمنة
      { roundId: 5, safeColumn: 3 }, // الخانة 4 آمنة
      { roundId: 6, safeColumn: 2 },
      { roundId: 7, safeColumn: 0 },
      { roundId: 8, safeColumn: 4 },
      { roundId: 9, safeColumn: 1 },
      { roundId: 10, safeColumn: 3 },
    ];

    res.json({ success: true, rounds });
  } catch (e) {
    next(e);
  }
});