"use client";

import { FormEvent, useEffect, useState } from "react";
import styles from "./page.module.css";

type Monitor = { id: string; url: string; interval_seconds: number; created_at: string };

const intervals = [
  { value: 5, label: "Каждые 5 секунд" },
  { value: 60, label: "Каждую минуту" },
  { value: 3600, label: "Каждый час" },
];

function intervalLabel(seconds: number) {
  return intervals.find((interval) => interval.value === seconds)?.label ?? `Каждые ${seconds} секунд`;
}

export default function DashboardPage() {
  const [monitors, setMonitors] = useState<Monitor[]>([]);
  const [url, setURL] = useState("");
  const [intervalSeconds, setIntervalSeconds] = useState(60);
  const [isFormOpen, setIsFormOpen] = useState(false);
  const [isLoading, setIsLoading] = useState(true);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    async function loadMonitors() {
      try {
        const response = await fetch("/api/monitors", { cache: "no-store" });
        const payload = await response.json().catch(() => ({}));
        if (!response.ok) {
          setError(payload.error?.message ?? "Не удалось загрузить точки мониторинга.");
          return;
        }
        setMonitors(payload.monitors ?? []);
      } catch {
        setError("Не удалось загрузить точки мониторинга.");
      } finally {
        setIsLoading(false);
      }
    }
    void loadMonitors();
  }, []);

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError("");
    setIsSubmitting(true);
    try {
      const response = await fetch("/api/monitors", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ url, interval_seconds: intervalSeconds }) });
      const payload = await response.json().catch(() => ({}));
      if (!response.ok) {
        setError(payload.error?.message ?? "Не удалось создать точку мониторинга.");
        return;
      }
      setMonitors((current) => [payload.monitor, ...current]);
      setURL("");
      setIntervalSeconds(60);
      setIsFormOpen(false);
    } catch {
      setError("Не удалось соединиться с сервисом.");
    } finally {
      setIsSubmitting(false);
    }
  }

  return (
    <section className={styles.dashboard}>
      <div className={styles.heading}>
        <div><p>Мониторинг</p><h1>Ваши проверки</h1><span>Добавьте сайт, чтобы создать точку мониторинга.</span></div>
        <button className={styles.addButton} type="button" onClick={() => { setError(""); setIsFormOpen(true); }}>Добавить сайт</button>
      </div>
      {isFormOpen && <form className={styles.form} onSubmit={submit}>
        <div className={styles.formHeading}><h2>Новая точка мониторинга</h2><button className={styles.closeButton} type="button" onClick={() => setIsFormOpen(false)} aria-label="Закрыть форму">×</button></div>
        <label>URL сайта<input type="url" value={url} onChange={(event) => setURL(event.target.value)} placeholder="https://example.com" required autoFocus /></label>
        <label>Частота опроса<select value={intervalSeconds} onChange={(event) => setIntervalSeconds(Number(event.target.value))}>{intervals.map((interval) => <option key={interval.value} value={interval.value}>{interval.label}</option>)}</select></label>
        {error && <p className={styles.error} role="alert">{error}</p>}
        <div className={styles.actions}><button className={styles.cancelButton} type="button" onClick={() => setIsFormOpen(false)}>Отмена</button><button className={styles.createButton} type="submit" disabled={isSubmitting}>{isSubmitting ? "Создаём…" : "Создать"}</button></div>
      </form>}
      {!isFormOpen && error && <p className={styles.error} role="alert">{error}</p>}
      {isLoading ? <p className={styles.muted}>Загружаем точки мониторинга…</p> : monitors.length === 0 ? <div className={styles.empty}><h2>Пока нет сайтов</h2><p>Создайте первую точку мониторинга — мы сохраним URL и выбранную частоту.</p></div> : <ul className={styles.monitorList} aria-label="Точки мониторинга">{monitors.map((monitor) => <li key={monitor.id} className={styles.monitor}><div><strong>{monitor.url}</strong><span>Создана {new Intl.DateTimeFormat("ru-RU", { dateStyle: "medium", timeStyle: "short" }).format(new Date(monitor.created_at))}</span></div><span className={styles.interval}>{intervalLabel(monitor.interval_seconds)}</span></li>)}</ul>}
    </section>
  );
}
