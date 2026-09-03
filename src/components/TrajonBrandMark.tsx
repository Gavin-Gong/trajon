import brandIcon from "../../src-tauri/icons/icon.png";

export function TrajonBrandMark() {
  return (
    <img
      data-testid="trading-journal-brand-mark"
      src={brandIcon}
      alt="Trajon"
      width={34}
      height={34}
      aria-hidden="true"
    />
  );
}
