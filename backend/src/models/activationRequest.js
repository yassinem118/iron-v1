import mongoose from 'mongoose';

const activationRequestSchema = new mongoose.Schema({
  user: { type: mongoose.Schema.Types.ObjectId, ref: 'User', required: true, index: true },
  accountId: { type: String, required: true, maxlength: 64 },
  status: { type: String, enum: ['pending', 'approved', 'rejected'], default: 'pending', index: true },
  note: { type: String, maxlength: 300, default: '' },
  requestedAt: { type: Date, default: Date.now },
  reviewedAt: { type: Date, default: null },
}, { timestamps: false, versionKey: false });
export const ActivationRequest = mongoose.model('ActivationRequest', activationRequestSchema);
