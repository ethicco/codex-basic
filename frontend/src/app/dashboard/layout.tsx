"use client";
/* eslint-disable @next/next/no-img-element -- avatar endpoint requires the browser session cookie */

import { ReactNode, useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import styles from "./layout.module.css";

type User = { id: string; email: string };

export default function DashboardLayout({ children }: { children: ReactNode }) {
  const router = useRouter();
  const [user, setUser] = useState<User | null>(null);
  const [profileName, setProfileName] = useState<string | null>(null);
  const [avatarURL, setAvatarURL] = useState<string | null>(null);
  const [menuOpen, setMenuOpen] = useState(false);
  const [loggingOut, setLoggingOut] = useState(false);

  useEffect(() => {
    async function loadSession() {
      const response = await fetch("/api/auth/session", { cache: "no-store" });
      if (!response.ok) {
        router.replace("/");
        return;
      }
      const payload = await response.json();
      setUser(payload.user);
    }
    void loadSession();
  }, [router]);

  useEffect(() => {
    if (!user) return;

    async function loadProfileName() {
      const response = await fetch("/api/profile", { cache: "no-store" });
      if (response.status === 404) {
        setProfileName(null);
        setAvatarURL(null);
        return;
      }
      if (!response.ok) return;
      const payload = await response.json();
      setProfileName(payload.profile.name);
      setAvatarURL(payload.profile.avatar_url ? `${payload.profile.avatar_url}?v=${payload.profile.updated_at}` : null);
    }

    void loadProfileName();
    window.addEventListener("profile-updated", loadProfileName);
    return () => window.removeEventListener("profile-updated", loadProfileName);
  }, [user]);

  async function logout() {
    setLoggingOut(true);
    await fetch("/api/auth/logout", { method: "POST" });
    router.replace("/");
    router.refresh();
  }

  if (!user) {
    return <div className={styles.loading}>Загружаем сессию…</div>;
  }

  const displayName = profileName ?? user.email;

  return (
    <div className={styles.shell}>
      <header className={styles.header}>
        <Link className={styles.logo} href="/dashboard">UPTIME</Link>
        <div className={styles.profile}>
          <button className={styles.profileButton} type="button" onClick={() => setMenuOpen((open) => !open)} aria-expanded={menuOpen} aria-haspopup="menu">
            {avatarURL ? <img className={styles.avatarImage} src={avatarURL} alt="Аватар" /> : <span className={styles.avatar}>{displayName.slice(0, 1).toUpperCase()}</span>}
            <span className={styles.email}>{displayName}</span>
          </button>
          {menuOpen && (
            <div className={styles.menu} role="menu">
              <div className={styles.profileDetails}>
                <strong>{displayName}</strong>
                <span>{user.email}</span>
              </div>
              <Link className={styles.profileLink} href="/profile" role="menuitem" onClick={() => setMenuOpen(false)}>
                Открыть профиль
              </Link>
              <button type="button" onClick={logout} disabled={loggingOut} role="menuitem">
                {loggingOut ? "Выходим…" : "Выйти"}
              </button>
            </div>
          )}
        </div>
      </header>
      <main className={styles.content}>{children}</main>
    </div>
  );
}
