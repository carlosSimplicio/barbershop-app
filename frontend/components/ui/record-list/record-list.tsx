import type { ComponentPropsWithRef, Key, ReactNode } from "react";

import styles from "./record-list.module.css";

export type RecordListProps<T> = Omit<
  ComponentPropsWithRef<"div">,
  "children"
> & {
  emptyMessage: string;
  getKey: (item: T) => Key;
  items: readonly T[];
  label: string;
  renderItem: (item: T) => ReactNode;
};

export function RecordList<T>({
  emptyMessage,
  getKey,
  items,
  label,
  renderItem,
  className,
  ...props
}: RecordListProps<T>) {
  const classes = className
    ? `${styles.container} ${className}`
    : styles.container;

  return (
    <div {...props} className={classes}>
      {items.length > 0 ? (
        <ul aria-label={label} className={styles.list}>
          {items.map((item) => (
            <li className={styles.item} key={getKey(item)}>
              {renderItem(item)}
            </li>
          ))}
        </ul>
      ) : (
        <p aria-live="polite" className={styles.empty} role="status">
          {emptyMessage}
        </p>
      )}
    </div>
  );
}
