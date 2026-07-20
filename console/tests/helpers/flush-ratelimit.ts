import { execSync } from 'child_process';

/**
 * Flush Redis rate limits before test runs.
 * Called in beforeAll of each test describe block.
 */
export async function flushRateLimits() {
  try {
    execSync('kubectl exec deploy/ggid-redis -n ggid -- redis-cli FLUSHALL', {
      stdio: 'pipe',
      timeout: 5000,
    });
  } catch {
    // Not in k8s environment — skip
  }
}
