import type { MouseEvent } from 'react';

interface RetryButtonProps {
  onRetry: () => void;
  disabled?: boolean;
  attempt?: number;
}

export function RetryButton({ onRetry, disabled = false, attempt = 0 }: RetryButtonProps) {
  const handleClick = (event: MouseEvent<HTMLButtonElement>) => {
    event.preventDefault();
    onRetry();
  };

  return (
    <button
      type="button"
      className="btn btn-secondary"
      aria-label="Retry failed request"
      disabled={disabled}
      onClick={handleClick}
    >
      {attempt > 0 ? `Retry (${attempt})` : 'Retry'}
    </button>
  );
}
