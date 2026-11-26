import type { WeeklySchedule, WorkPeriod } from "./weekly-schedule-api";

type Validation =
  { valid: true; periods: WorkPeriod[] } | { valid: false; message: string };

export const WEEKDAYS = [
  "Segunda-feira",
  "Terça-feira",
  "Quarta-feira",
  "Quinta-feira",
  "Sexta-feira",
  "Sábado",
  "Domingo",
] as const;

const maxPeriods = WEEKDAYS.length * 24 * 60;
const localTimePattern = /^(?:[01][0-9]|2[0-3]):[0-5][0-9]$/;
const midnightEnd = "24:00";

export function compareWorkPeriods(left: WorkPeriod, right: WorkPeriod) {
  return (
    left.weekday - right.weekday ||
    left.start_time.localeCompare(right.start_time)
  );
}

export function validateWorkPeriods(input: readonly WorkPeriod[]): Validation {
  if (input.length > maxPeriods) {
    return {
      valid: false,
      message: `A semana permite até ${maxPeriods.toLocaleString("pt-BR")} períodos.`,
    };
  }
  const periods: WorkPeriod[] = [];
  for (const { weekday, start_time, end_time } of input) {
    const day = WEEKDAYS[weekday - 1];
    if (!day) {
      return { valid: false, message: "Confira os dias da semana." };
    }
    if (
      !isLocalTime(start_time) ||
      (!isLocalTime(end_time) && end_time !== midnightEnd)
    ) {
      return {
        valid: false,
        message: `${day}: informe os horários no formato HH:mm. Use 24:00 somente no fim.`,
      };
    }
    if (start_time >= end_time) {
      return {
        valid: false,
        message: `${day}: o início do período precisa ser anterior ao fim.`,
      };
    }
    periods.push({ weekday, start_time, end_time });
  }
  periods.sort(compareWorkPeriods);
  for (let index = 1; index < periods.length; index++) {
    const previous = periods[index - 1];
    const current = periods[index];
    if (
      current.weekday === previous.weekday &&
      current.start_time < previous.end_time
    ) {
      return {
        valid: false,
        message: `${WEEKDAYS[current.weekday - 1]}: há sobreposição de períodos no mesmo dia.`,
      };
    }
  }
  return { valid: true, periods };
}

export function parseWeeklySchedule(
  value: unknown,
  professionalId: string,
): WeeklySchedule | null {
  if (
    typeof value !== "object" ||
    value === null ||
    !("professional_id" in value) ||
    value.professional_id !== professionalId ||
    !("time_zone" in value) ||
    typeof value.time_zone !== "string" ||
    !isTimeZone(value.time_zone) ||
    !("can_write" in value) ||
    typeof value.can_write !== "boolean" ||
    !("periods" in value) ||
    !Array.isArray(value.periods) ||
    value.periods.length > maxPeriods ||
    !value.periods.every(isWorkPeriod)
  ) {
    return null;
  }
  const result = validateWorkPeriods(value.periods);
  if (!result.valid) return null;
  return {
    professional_id: professionalId,
    time_zone: value.time_zone,
    can_write: value.can_write,
    periods: result.periods,
  };
}

export function sameWorkPeriods(left: WorkPeriod[], right: WorkPeriod[]) {
  return (
    left.length === right.length &&
    left.every(
      (period, index) =>
        period.weekday === right[index].weekday &&
        period.start_time === right[index].start_time &&
        period.end_time === right[index].end_time,
    )
  );
}

function isLocalTime(value: string) {
  return value.length === midnightEnd.length && localTimePattern.test(value);
}

function isTimeZone(value: string) {
  // Intl also accepts numeric offsets, while the API returns IANA names.
  if (!/^[A-Za-z]/.test(value)) return false;
  try {
    new Intl.DateTimeFormat("pt-BR", { timeZone: value });
    return true;
  } catch {
    return false;
  }
}

function isWorkPeriod(value: unknown): value is WorkPeriod {
  return (
    typeof value === "object" &&
    value !== null &&
    "weekday" in value &&
    typeof value.weekday === "number" &&
    Number.isInteger(value.weekday) &&
    "start_time" in value &&
    typeof value.start_time === "string" &&
    "end_time" in value &&
    typeof value.end_time === "string"
  );
}
