export type Monitor = { id: string; url: string; interval_seconds: number; created_at: string };

export const monitorIntervals = [
  { value: 5, label: "Каждые 5 секунд" },
  { value: 60, label: "Каждую минуту" },
  { value: 3600, label: "Каждый час" },
];

export function monitorIntervalLabel(seconds: number) {
  return monitorIntervals.find((interval) => interval.value === seconds)?.label ?? `Каждые ${seconds} секунд`;
}
