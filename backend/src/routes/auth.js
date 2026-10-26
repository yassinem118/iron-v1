import express from 'express';

// 1. تعريف الـ Router بالاسم الصحيح authRouter
const authRouter = express.Router();

// 2. الـ Route ديالك
authRouter.post('/connect', async (req, res, next) => {
  try {
    const parsed = z.object({
      accountId: z.string().trim().min(1).max(128),
      password: z.string().min(1).max(128),
    }).safeParse(req.body);

    if (!parsed.success) {
      return res.status(400).json({
        success: false,
        message: 'Invalid account ID or password',
        redirect: false,
      });
    }

    const { accountId, password } = parsed.data;

    let user = await User.findOne({ accountId }).select('+passwordHash');
    let created = false;

    if (!user) {
      const passwordHash = await argon2.hash(password, {
        type: argon2.argon2id,
      });

      try {
        user = await User.create({ accountId, passwordHash });
        created = true;
        audit('account.registered', { accountId });
      } catch (error) {
        // Handle concurrent first-time registration.
        if (error?.code !== 11000) throw error;

        user = await User.findOne({ accountId })
          .select('+passwordHash');
      }
    }

    if (
      !user ||
      !user.active ||
      !(await argon2.verify(user.passwordHash, password))
    ) {
      audit('account.connect_failed', { accountId });

      return res.status(401).json({
        success: false,
        message: 'Invalid account ID or password',
        redirect: false,
      });
    }

    user.lastLoginAt = new Date();
    await user.save();

    const token = jwt.sign(
      { sub: user.id },
      env.jwtSecret,
      {
        expiresIn: '30m',
        issuer: 'iron-v1',
        audience: 'iron-v1-client',
      }
    );

    res.cookie('iron_session', token, {
      httpOnly: true,
      secure: env.production,
      sameSite: 'strict',
      maxAge: 30 * 60 * 1000,
    });

    audit('account.connect_success', {
      accountId: user.accountId,
      created,
    });

    return res.status(200).json({
      success: true,
      message: 'Credentials saved successfully',
      redirect: true,
    });
  } catch (error) {
    console.error('POST /api/connect failed:', error);
    return next(error);
  }
});

// 3. التصدير النظيف بنفس الاسم
export { authRouter };