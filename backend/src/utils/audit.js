// Keep audit events free of passwords, tokens, and other secrets.
export function audit(event, metadata = {}) {
  console.log(JSON.stringify({ time: new Date().toISOString(), event, ...metadata }));
}
