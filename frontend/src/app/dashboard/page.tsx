"use client";

import { FormEvent, useEffect, useState } from "react";
import { Monitor, monitorIntervalLabel, monitorIntervals } from "@/lib/monitors";
import styles from "./page.module.css";

export default function DashboardPage() {
  const [monitors, setMonitors] = useState<Monitor[]>([]);
  const [nextCursor, setNextCursor] = useState("");
  const [url, setURL] = useState("");
  const [intervalSeconds, setIntervalSeconds] = useState(60);
  const [editingMonitor, setEditingMonitor] = useState<Monitor | null>(null);
  const [isFormOpen, setIsFormOpen] = useState(false);
  const [isLoading, setIsLoading] = useState(true);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [pendingMonitorID, setPendingMonitorID] = useState("");
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
        setNextCursor(payload.next_cursor ?? "");
      } catch {
        setError("Не удалось загрузить точки мониторинга.");
      } finally {
        setIsLoading(false);
      }
    }
    void loadMonitors();
  }, []);

  function closeForm() {
    setIsFormOpen(false);
    setEditingMonitor(null);
    setURL("");
    setIntervalSeconds(60);
  }

  function openCreateForm() {
    setError("");
    setEditingMonitor(null);
    setURL("");
    setIntervalSeconds(60);
    setIsFormOpen(true);
  }

  function openEditForm(monitor: Monitor) {
    setError("");
    setEditingMonitor(monitor);
    setURL(monitor.url);
    setIntervalSeconds(monitor.interval_seconds);
    setIsFormOpen(true);
  }

  async function loadMore() {
    if (!nextCursor) return;
    setIsLoading(true);
    try {
      const response = await fetch(`/api/monitors?cursor=${encodeURIComponent(nextCursor)}`, { cache: "no-store" });
      const payload = await response.json().catch(() => ({}));
      if (!response.ok) {
        setError(payload.error?.message ?? "Не удалось загрузить точки мониторинга.");
        return;
      }
      setMonitors((current) => [...current, ...(payload.monitors ?? [])]);
      setNextCursor(payload.next_cursor ?? "");
    } catch {
      setError("Не удалось загрузить точки мониторинга.");
    } finally {
      setIsLoading(false);
    }
  }

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError("");
    setIsSubmitting(true);
    const isEditing = editingMonitor !== null;
    const endpoint = isEditing ? `/api/monitors/${editingMonitor.id}` : "/api/monitors";
    try {
      const response = await fetch(endpoint, {
        method: isEditing ? "PATCH" : "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ url, interval_seconds: intervalSeconds }),
      });
      const payload = await response.json().catch(() => ({}));
      if (!response.ok) {
        setError(payload.error?.message ?? (isEditing ? "Не удалось сохранить точку мониторинга." : "Не удалось создать точку мониторинга."));
        return;
      }
      if (isEditing) {
        setMonitors((current) => current.map((monitor) => monitor.id === payload.monitor.id ? payload.monitor : monitor));
      } else {
        setMonitors((current) => [payload.monitor, ...current]);
      }
      closeForm();
    } catch {
      setError("Не удалось соединиться с сервисом.");
    } finally {
      setIsSubmitting(false);
    }
  }

  async function deleteMonitor(monitor: Monitor) {
    if (!window.confirm(`Удалить сайт ${monitor.url}?`)) return;
    setError("");
    setPendingMonitorID(monitor.id);
    try {
      const response = await fetch(`/api/monitors/${monitor.id}`, { method: "DELETE" });
      if (!response.ok) {
        const payload = await response.json().catch(() => ({}));
        setError(payload.error?.message ?? "Не удалось удалить точку мониторинга.");
        return;
      }
      setMonitors((current) => current.filter((entry) => entry.id !== monitor.id));
      if (editingMonitor?.id === monitor.id) closeForm();
    } catch {
      setError("Не удалось соединиться с сервисом.");
    } finally {
      setPendingMonitorID("");
    }
  }

  const formTitle = editingMonitor ? "Редактирование точки мониторинга" : "Новая точка мониторинга";
  const submitLabel = editingMonitor ? "Сохранить" : "Создать";

  return (
    <section className={styles.dashboard}>
      <div className={styles.heading}>
        <div><p>Мониторинг</p><h1>Ваши проверки</h1><span>Добавьте сайт, чтобы создать точку мониторинга.</span></div>
        <button className={styles.addButton} type="button" onClick={openCreateForm} disabled={isSubmitting}>Добавить сайт</button>
      </div>
      {isFormOpen && <form className={styles.form} onSubmit={submit}>
        <div className={styles.formHeading}><h2>{formTitle}</h2><button className={styles.closeButton} type="button" onClick={closeForm} aria-label="Закрыть форму" disabled={isSubmitting}>×</button></div>
        <label>URL сайта<input type="url" value={url} onChange={(event) => setURL(event.target.value)} placeholder="https://example.com" required autoFocus disabled={isSubmitting} /></label>
        <label>Частота опроса<select value={intervalSeconds} onChange={(event) => setIntervalSeconds(Number(event.target.value))} disabled={isSubmitting}>{monitorIntervals.map((interval) => <option key={interval.value} value={interval.value}>{interval.label}</option>)}</select></label>
        {error && <p className={styles.error} role="alert">{error}</p>}
        <div className={styles.actions}><button className={styles.cancelButton} type="button" onClick={closeForm} disabled={isSubmitting}>Отмена</button><button className={styles.createButton} type="submit" disabled={isSubmitting}>{isSubmitting ? "Сохраняем…" : submitLabel}</button></div>
      </form>}
      {!isFormOpen && error && <p className={styles.error} role="alert">{error}</p>}
      {isLoading && monitors.length === 0 ? <p className={styles.muted}>Загружаем точки мониторинга…</p> : monitors.length === 0 ? <div className={styles.empty}><h2>Пока нет сайтов</h2><p>Создайте первую точку мониторинга — мы сохраним URL и выбранную частоту.</p></div> : <><ul className={styles.monitorList} aria-label="Точки мониторинга">{monitors.map((monitor) => <li key={monitor.id} className={styles.monitor}><div><strong>{monitor.url}</strong><span>Создана {new Intl.DateTimeFormat("ru-RU", { dateStyle: "medium", timeStyle: "short" }).format(new Date(monitor.created_at))}</span></div><div className={styles.monitorActions}><span className={styles.interval}>{monitorIntervalLabel(monitor.interval_seconds)}</span><button className={styles.editButton} type="button" onClick={() => openEditForm(monitor)} disabled={isSubmitting || pendingMonitorID !== ""}>Редактировать</button><button className={styles.deleteButton} type="button" onClick={() => void deleteMonitor(monitor)} disabled={isSubmitting || pendingMonitorID !== ""}>{pendingMonitorID === monitor.id ? "Удаляем…" : "Удалить"}</button></div></li>)}</ul>{nextCursor && <button className={styles.loadMoreButton} type="button" onClick={loadMore} disabled={isLoading || isSubmitting || pendingMonitorID !== ""}>{isLoading ? "Загружаем…" : "Показать ещё"}</button>}</>}
    </section>
  );
}
