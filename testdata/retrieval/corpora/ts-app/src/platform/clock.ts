export interface Clock {
  now(): Date;
}

export const systemClock: Clock = {
  now: () => new Date(),
};
