const express = require('express');
const router = express.Router();

// نموذج قاعدة البيانات للجولات الحية المرتبطة بـ Mise
const ActiveRoundModel = require('../models/ActiveRound'); 

// مسار جلب نتائج الجولة الحقيقية فقط
router.get('/rounds', async (req, res) => {
  try {
    // جلب آخر جولة حقيقية مسجلة من السيرفر أو البوت (مرتبطة بـ Mise الفعلي للزبون)
    const activeRound = await ActiveRoundModel.findOne().sort({ createdAt: -1 });

    // الشرط الصارم: ممنوع النتائج الوهمية أو المحاكاة!
    if (!activeRound || !activeRound.isLive || !activeRound.roundsData) {
      return res.status(404).json({
        success: false,
        message: 'لم يتم رصد أي رهان نشط. قم بوضع Mise في لعبة Melbet أولاً!'
      });
    }

    // إرجاع النتائج بالصيغة التي ينتظرها تطبيق الفلاتر تماماً
    return res.status(200).json({
      success: true,
      rounds: activeRound.roundsData // مصفوفة تحتوي على الجولة والخانة الآمنة لكل صف
    });

  } catch (error) {
    console.error('Error fetching Apple Fortune rounds:', error);
    return res.status(500).json({
      success: false,
      message: 'خطأ في الاتصال بالخادم الرئيسي'
    });
  }
});

module.exports = router;