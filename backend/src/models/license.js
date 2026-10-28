import mongoose from 'mongoose';

const licenseSchema = new mongoose.Schema({
  user: { type: mongoose.Schema.Types.ObjectId, ref: 'User', required: true, unique: true, index: true },
  status: { type: String, enum: ['trial', 'pending', 'active', 'expired', 'revoked'], default: 'trial', index: true },
  trialRemaining: { type: Number, min: 0, max: 2, default: 2 },
  activationCodeHash: { type: String, select: false },
  licenseDurationDays: { type: Number, min: 1, max: 365, default: 15 },
  startsAt: { type: Date, default: null },
  expiresAt: { type: Date, default: null, index: true },
  renewalRequestedAt: { type: Date, default: null },
  updatedByAdminAt: { type: Date, default: null },
  activatedAt: { type: Date, default: null },
}, { timestamps: true, versionKey: false });

licenseSchema.index({ status: 1, expiresAt: 1 });
export const License = mongoose.model('License', licenseSchema);
