import mongoose from 'mongoose';

const userSchema = new mongoose.Schema({
  // This is an Iron v1 username/account ID, not a third-party gambling login.
  accountId: { type: String, required: true, trim: true, minlength: 3, maxlength: 64, unique: true, index: true },
  passwordHash: { type: String, required: true, select: false },
  active: { type: Boolean, default: true, index: true },
  createdAt: { type: Date, default: Date.now, immutable: true },
  lastLoginAt: { type: Date, default: null },
}, { timestamps: false, versionKey: false });

export const User = mongoose.model('User', userSchema);
