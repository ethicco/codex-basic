"use client";
/* eslint-disable @next/next/no-img-element -- avatar endpoint requires the browser session cookie */

import { ChangeEvent, FormEvent, useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import styles from "../profile.module.css";

export default function EditProfilePage() {
  const router = useRouter();
  const [name, setName] = useState("");
  const [loading, setLoading] = useState(true);
  const [pending, setPending] = useState(false);
  const [error, setError] = useState("");
  const [avatarURL, setAvatarURL] = useState("");
  const [avatarFile, setAvatarFile] = useState<File | null>(null);
  const [avatarPreview, setAvatarPreview] = useState("");

  useEffect(() => {
    async function loadProfile() {
      const response = await fetch("/api/profile", { cache: "no-store" });
      if (response.status === 404) {
        router.replace("/profile");
        return;
      }
      if (!response.ok) {
        router.replace("/");
        return;
      }
      const payload = await response.json();
      setName(payload.profile.name);
      setAvatarURL(payload.profile.avatar_url ? `${payload.profile.avatar_url}?v=${payload.profile.updated_at}` : "");
      setLoading(false);
    }
    void loadProfile();
  }, [router]);

  function selectAvatar(event: ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0] ?? null;
    if (!file) return;
    if (!file.type.startsWith("image/")) {
      setError("Выберите файл изображения.");
      return;
    }
    setError("");
    setAvatarFile(file);
    setAvatarPreview(URL.createObjectURL(file));
  }

  async function saveProfile(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError("");
    setPending(true);
    try {
      const response = await fetch("/api/profile", {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ name }),
      });
      const payload = await response.json().catch(() => ({}));
      if (!response.ok) {
        setError(payload.error?.message ?? "Не удалось сохранить профиль.");
        return;
      }
      if (avatarFile) {
        const formData = new FormData();
        formData.set("avatar", avatarFile);
        const avatarResponse = await fetch("/api/profile/avatar", { method: "POST", body: formData });
        const avatarPayload = await avatarResponse.json().catch(() => ({}));
        if (!avatarResponse.ok) {
          setError(avatarPayload.error?.message ?? "Не удалось загрузить аватар.");
          return;
        }
        setAvatarURL(`${avatarPayload.profile.avatar_url}?v=${avatarPayload.profile.updated_at}`);
      }
      window.dispatchEvent(new Event("profile-updated"));
      router.replace("/profile");
      router.refresh();
    } catch {
      setError("Не удалось соединиться с сервисом профилей.");
    } finally {
      setPending(false);
    }
  }

  if (loading) return <p className={styles.loading}>Загружаем профиль…</p>;

  return (
    <section className={styles.card}>
      <p className={styles.eyebrow}>РЕДАКТИРОВАНИЕ ПРОФИЛЯ</p>
      <h1>Ваше имя</h1>
      <form className={styles.form} onSubmit={saveProfile}>
        <div className={styles.avatarEditor}>
          {avatarPreview || avatarURL ? <img className={styles.profileAvatar} src={avatarPreview || avatarURL} alt="Текущий аватар" /> : <span className={styles.profileAvatarFallback}>{name.slice(0, 1).toUpperCase() || "?"}</span>}
          <label className={styles.avatarUpload}>
            Загрузить аватар
            <input type="file" accept="image/jpeg,image/png,image/gif,image/webp" onChange={selectAvatar} disabled={pending} />
          </label>
          <span className={styles.avatarHint}>JPEG, PNG, GIF или WebP, до 5 МБ</span>
        </div>
        <label>
          Имя
          <input value={name} onChange={(event) => setName(event.target.value)} minLength={1} maxLength={100} autoComplete="name" required />
        </label>
        {error && <p className={styles.error} role="alert">{error}</p>}
        <div className={styles.actions}>
          <button className={styles.secondary} type="button" onClick={() => router.back()} disabled={pending}>Отмена</button>
          <button className={styles.primary} type="submit" disabled={pending}>{pending ? "Сохраняем…" : "Сохранить"}</button>
        </div>
      </form>
    </section>
  );
}
