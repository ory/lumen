import { chargeDeposit, refundDeposit } from './payments';
import { holdRoom } from './reservation.js';
import sendConfirmation from './mailer';

export interface BookingRequest {
  guestId: string;
  roomId: string;
  nights: number;
}

// Confirms a booking once the deposit is captured.
export async function confirmBooking(req: BookingRequest): Promise<string> {
  const depositId = await chargeDeposit(req.guestId, req.nights * 2500);
  await sendConfirmation(req.guestId, depositId);
  return depositId;
}

// Places a tentative hold while the guest is still on the checkout page.
export async function startBooking(req: BookingRequest): Promise<void> {
  await holdRoom(req.roomId, req.nights);
}

// Returns the deposit when the guest cancels before the free-cancellation cutoff.
export async function cancelBooking(depositId: string): Promise<void> {
  await refundDeposit(depositId);
}
