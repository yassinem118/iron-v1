import test from 'node:test';
import assert from 'node:assert/strict';
import { calibrate, summarizeOutcomes } from './calibrator.js';

test('empty dataset has no fabricated confidence', () => {
  const result = calibrate([]);
  assert.equal(result.samples, 0);
  assert.equal(result.calibratedRate, null);
  assert.equal(result.confidence, 0);
});

test('summary and smoothed calibration use observed outcomes', () => {
  const rows = [
    { predictedProbability: 0.8, outcome: 'success' },
    { predictedProbability: 0.7, outcome: 'failure' },
  ];
  const s = summarizeOutcomes(rows);
  assert.equal(s.samples, 2);
  assert.equal(s.observedSuccessRate, 0.5);
  assert.ok(s.brierScore >= 0 && s.brierScore <= 1);
  const c = calibrate(rows);
  assert.equal(c.calibratedRate, 0.5);
  assert.equal(c.confidence, 0.2);
});
