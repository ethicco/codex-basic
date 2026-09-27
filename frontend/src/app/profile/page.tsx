"use client";
/* eslint-disable @next/next/no-img-element -- avatar endpoint requires the browser session cookie */

import Link from "next/link";
import { FormEvent, useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import styles from "./profile.module.css";

type Profile = { name: string; avatar_url: string; created_at: string; updated_at: string };

export default function ProfilePage() {
  const router = useRouter();
  const [profile, setProfile] = useState<Profile | null>(null);
  const [missing, setMissing] = useState(false);
  const [name, setName] = useState("");
  const [error, setError] = useState("");
  const [pending, setPending] = useState(false);

  useEffect(() => {
    async function loadProfile() {
      const response = await fetch("/api/profile", { cache: "no-store" });
      if (response.status === 404) {
        setMissing(true);
        return;
      }
      if (!response.ok) {
        router.replace("/");
        return;
      }
      const payload = await response.json();
      setProfile(payload.profile);
    }
    void loadProfile();
  }, [router]);

  async function createProfile(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError("");
    setPending(true);
    try {
      const response = await fetch("/api/profile", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ name }),
      });
      const payload = await response.json().catch(() => ({}));
      if (!response.ok) {
        setError(payload.error?.message ?? "Не удалось создать профиль.");
        return;
      }
      setProfile(payload.profile);
      setMissing(false);
      window.dispatchEvent(new Event("profile-updated"));
    } catch {
      setError("Не удалось соединиться с сервисом профилей.");
    } finally {
      setPending(false);
    }
  }

  if (!profile && !missing) return <p className={styles.loading}>Загружаем профиль…</p>;

  return (
    <section className={styles.card}>
      {profile ? (
        <>
          <p className={styles.eyebrow}>ПРОФИЛЬ</p>
          {profile.avatar_url && <img className={styles.profileAvatar} src={`${profile.avatar_url}?v=${profile.updated_at}`} alt={`Аватар ${profile.name}`} />}
          <h1>{profile.name}</h1>
          <p className={styles.meta}>Измените имя или загрузите аватар в настройках профиля.</p>
          <Link className={styles.primary} href="/profile/edit">Редактировать профиль</Link>
        </>
      ) : (
        <>
          <p className={styles.eyebrow}>НОВЫЙ ПРОФИЛЬ</p>
          <h1>Создайте профиль</h1>
          <p className={styles.meta}>Укажите имя, которое будет отображаться в вашем профиле.</p>
          <form className={styles.form} onSubmit={createProfile}>
            <label>
              Имя
              <input value={name} onChange={(event) => setName(event.target.value)} minLength={1} maxLength={100} autoComplete="name" required />
            </label>
            {error && <p className={styles.error} role="alert">{error}</p>}
            <button className={styles.primary} type="submit" disabled={pending}>{pending ? "Сохраняем…" : "Создать профиль"}</button>
          </form>
        </>
      )}
    </section>
  );
}
