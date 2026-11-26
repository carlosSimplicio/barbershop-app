import type { WeeklySchedule, WorkPeriod } from "./weekly-schedule-api";

export type WorkPeriodTimeField = "start_time" | "end_time";

type Validation =
  | { valid: true; periods: WorkPeriod[] }
  | {
      valid: false;
      message: string;
      target?: { period: WorkPeriod; field: WorkPeriodTimeField };
    };

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
  for (const period of input) {
    const { weekday, start_time, end_time } = period;
    const day = WEEKDAYS[weekday - 1];
    if (!day) {
      return { valid: false, message: "Confira os dias da semana." };
    }
    const validStart = isLocalTime(start_time);
    if (!validStart || (!isLocalTime(end_time) && end_time !== midnightEnd)) {
      return {
        valid: false,
        message: `${day}: informe os horários no formato HH:mm. Use 24:00 somente no fim.`,
        target: { period, field: validStart ? "end_time" : "start_time" },
      };
    }
    if (start_time >= end_time) {
      return {
        valid: false,
        message: `${day}: o início do período precisa ser anterior ao fim.`,
        target: { period, field: "end_time" },
      };
    }
  }
  const periods = [...input].sort(compareWorkPeriods);
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
        target: { period: current, field: "start_time" },
      };
    }
  }
  return {
    valid: true,
    periods: periods.map(({ weekday, start_time, end_time }) => ({
      weekday,
      start_time,
      end_time,
    })),
  };
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
    value.time_zone.trim() === "" ||
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

export function sameWorkPeriods(
  left: readonly WorkPeriod[],
  right: readonly WorkPeriod[],
) {
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
