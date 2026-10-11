interface CloseButtonProps {
  onClose: () => void;
}

export function CloseButton({ onClose }: CloseButtonProps) {
  return (
    <button type="button" className="btn-icon" aria-label="Close dialog" onClick={onClose}>
      <span aria-hidden="true">&times;</span>
    </button>
  );
}
