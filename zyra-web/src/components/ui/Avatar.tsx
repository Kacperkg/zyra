import { useState } from "react";
import styles from "./Avatar.module.css";
export function Avatar({
  user,
  size = 32,
}: {
  user: { name: string; avatar_url?: string };
  size?: number;
}) {
  const [failedURL, setFailedURL] = useState("");
  const url = user.avatar_url || "";
  const initials =
    user.name
      .trim()
      .split(/\s+/)
      .slice(0, 2)
      .map((x) => x[0])
      .join("")
      .toUpperCase() || "?";
  return (
    <span
      className={styles.avatar}
      style={{ width: size, height: size }}
      aria-hidden="true"
    >
      {url.startsWith("https://") && failedURL !== url ? (
        <img
          src={url}
          alt=""
          referrerPolicy="no-referrer"
          onError={() => setFailedURL(url)}
        />
      ) : (
        initials
      )}
    </span>
  );
}
