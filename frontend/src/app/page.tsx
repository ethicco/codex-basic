"use client";

import { FormEvent, useState } from "react";
import { useRouter } from "next/navigation";
import styles from "./page.module.css";

type AuthMode = "login" | "register";

export default function Home() {
  const [mode, setMode] = useState<AuthMode>("login");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [confirmation, setConfirmation] = useState("");
  const [error, setError] = useState("");
  const [pending, setPending] = useState(false);
  const router = useRouter();
  const isRegister = mode === "register";

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError("");
    if (isRegister && password !== confirmation) {
      setError("Пароли не совпадают.");
      return;
    }
    setPending(true);
    try {
      const response = await fetch(`/api/auth/${mode}`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ email, password }),
      });
      const payload = await response.json().catch(() => ({}));
      if (!response.ok) {
        setError(payload.error?.message ?? "Не удалось выполнить запрос. Попробуйте ещё раз.");
        return;
      }
      setPassword("");
      setConfirmation("");
      router.replace("/dashboard");
      router.refresh();
    } catch {
      setError("Не удалось соединиться с сервисом авторизации.");
    } finally {
      setPending(false);
    }
  }

  function switchMode(nextMode: AuthMode) {
    setMode(nextMode);
    setError("");
  }

  return (
    <div className={styles.page}>
      <main className={styles.card}>
        <div className={styles.brand} aria-hidden="true">UP</div>
        <div className={styles.heading}>
          <p className={styles.eyebrow}>UPTIME</p>
          <h1>{isRegister ? "Создайте аккаунт" : "С возвращением"}</h1>
          <p>{isRegister ? "Начните следить за доступностью сервисов." : "Войдите, чтобы продолжить работу."}</p>
        </div>

        <div className={styles.tabs} role="tablist" aria-label="Авторизация">
          <button className={!isRegister ? styles.activeTab : ""} type="button" onClick={() => switchMode("login")} role="tab" aria-selected={!isRegister}>Вход</button>
          <button className={isRegister ? styles.activeTab : ""} type="button" onClick={() => switchMode("register")} role="tab" aria-selected={isRegister}>Регистрация</button>
        </div>

        <form className={styles.form} onSubmit={submit}>
          <label>
            Email
            <input type="email" value={email} onChange={(event) => setEmail(event.target.value)} placeholder="name@company.ru" autoComplete="email" required />
          </label>
          <label>
            Пароль
            <input type="password" value={password} onChange={(event) => setPassword(event.target.value)} placeholder="Минимум 12 символов" autoComplete={isRegister ? "new-password" : "current-password"} minLength={12} maxLength={128} required />
          </label>
          {isRegister && <label>Повторите пароль<input type="password" value={confirmation} onChange={(event) => setConfirmation(event.target.value)} autoComplete="new-password" minLength={12} maxLength={128} required /></label>}
          {error && <p className={styles.error} role="alert">{error}</p>}
          <button className={styles.submit} type="submit" disabled={pending}>{pending ? "Подождите…" : isRegister ? "Создать аккаунт" : "Войти"}</button>
        </form>
        <p className={styles.note}>Токены защищены HttpOnly session cookies и недоступны JavaScript.</p>
      </main>
    </div>
  );
}
