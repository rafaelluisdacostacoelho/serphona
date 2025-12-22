import { describe, expect, it } from 'vitest';
import { billingApi } from './billingApi';

describe('billingApi', () => {
  it('returns plans from mocked API without real HTTP', async () => {
    const plans = await billingApi.getPlans();
    expect(plans).toHaveLength(2);
    expect(plans[0].name).toBe('Basic');
  });
});
