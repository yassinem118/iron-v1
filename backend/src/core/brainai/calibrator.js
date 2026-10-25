// Transparent calibration of previously observed probabilities.
// This is not a hidden-state solver and cannot forecast random game outcomes.
export function summarizeOutcomes(rows) {
  const n = rows.length;
  if (!n) return { samples: 0, observedSuccessRate: null, brierScore: null, meanReportedProbability: null };
  let successes = 0, probSum = 0, brier = 0;
  for (const row of rows) {
    const y = row.outcome === 'success' ? 1 : 0;
    const p = Math.max(0, Math.min(1, Number(row.predictedProbability)));
    successes += y; probSum += p; brier += (p - y) ** 2;
  }
  return {
    samples: n,
    observedSuccessRate: successes / n,
    meanReportedProbability: probSum / n,
    brierScore: brier / n,
  };
}

// Reliability-weighted historical estimate with Laplace smoothing.
// A prior prevents extreme confidence from tiny samples.
export function calibrate(rows, priorStrength = 8) {
  const summary = summarizeOutcomes(rows);
  if (!summary.samples) return { ...summary, calibratedRate: null, confidence: 0 };
  const prior = 0.5;
  const calibratedRate = (rows.filter(r => r.outcome === 'success').length + priorStrength * prior) /
    (rows.length + priorStrength);
  const confidence = rows.length / (rows.length + priorStrength);
  return { ...summary, calibratedRate, confidence, priorStrength };
}
