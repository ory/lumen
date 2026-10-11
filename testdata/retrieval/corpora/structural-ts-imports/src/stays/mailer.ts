import { smtp } from '../smtp';

export default async function sendConfirmation(guestId: string, depositId: string): Promise<void> {
  await smtp.send({ to: guestId, template: 'booking-confirmed', vars: { depositId } });
}
