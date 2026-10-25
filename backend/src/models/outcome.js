import mongoose from 'mongoose';

const outcomeSchema = new mongoose.Schema({
  user: { type: mongoose.Schema.Types.ObjectId, ref: 'User', required: true, index: true },
  predictedProbability: { type: Number, required: true, min: 0, max: 1 },
  outcome: { type: String, enum: ['success', 'failure'], required: true },
  eventType: { type: String, maxlength: 40, default: 'observed' },
  durationMs: { type: Number, min: 0, max: 86400000 },
  createdAt: { type: Date, default: Date.now, immutable: true },
}, { versionKey: false });
outcomeSchema.index({ user: 1, createdAt: -1 });
export const Outcome = mongoose.model('Outcome', outcomeSchema);
