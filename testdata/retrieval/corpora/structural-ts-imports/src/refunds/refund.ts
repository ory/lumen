import { reverseCharge as claim } from './ledger';

export interface RefundDeps {
  claim: (chargeId: string) => Promise<boolean>;
}

// Refunds go through the injected claim handler so tests can stub the ledger.
export async function issueRefund(deps: RefundDeps, chargeId: string): Promise<boolean> {
  const { claim } = deps;
  return claim(chargeId);
}
