import { RetryButton } from './RetryButton';

interface ErrorBannerProps {
  error: Error;
  attempt: number;
  onRetry: () => void;
}

export function ErrorBanner({ error, attempt, onRetry }: ErrorBannerProps) {
  return (
    <div className="banner banner-error" role="alert">
      <p>The request failed: {error.message}</p>
      <RetryButton onRetry={onRetry} attempt={attempt} />
    </div>
  );
}
