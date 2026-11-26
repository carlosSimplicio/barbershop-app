import { memo, useLayoutEffect, useRef, type Ref } from "react";
import { Button, SelectField, TextField } from "@/components/ui";
import type { WorkPeriodTimeField } from "@/lib/professionals/weekly-schedule";
import type { WorkPeriod } from "@/lib/professionals/weekly-schedule-api";
import styles from "./weekly-schedule-editor.module.css";

export type DraftPeriod = WorkPeriod & { id: number };
export type PeriodFocus = {
  id: number;
  weekday: number;
  field: WorkPeriodTimeField;
};

type PeriodActions = {
  onEdit: (
    period: DraftPeriod,
    field: WorkPeriodTimeField,
    value: string,
  ) => void;
  onRemove: (period: DraftPeriod) => void;
};

type DayProps = PeriodActions & {
  day: string;
  weekday: number;
  periods: readonly DraftPeriod[];
  page: number;
  focusTarget: PeriodFocus | null;
  writable: boolean;
  submitting: boolean;
  onPageChange: (weekday: number, page: number) => void;
  onAdd: (weekday: number, page: number) => void;
};

type RowProps = PeriodActions & {
  period: DraftPeriod;
  day: string;
  number: number;
  writable: boolean;
  submitting: boolean;
  focusField?: WorkPeriodTimeField;
  focusRef?: Ref<HTMLInputElement>;
};

const timeFields = [
  { name: "start_time", label: "Início" },
  { name: "end_time", label: "Fim" },
] as const;

const periodsPerPage = 20;

export function pageOfPeriod(index: number) {
  return Math.floor(index / periodsPerPage);
}

export const WeeklyScheduleDay = memo(function WeeklyScheduleDay({
  day,
  weekday,
  periods,
  page,
  focusTarget,
  writable,
  submitting,
  onPageChange,
  onAdd,
  onEdit,
  onRemove,
}: DayProps) {
  const focusRef = useRef<HTMLInputElement>(null);
  useLayoutEffect(() => {
    if (focusTarget) focusRef.current?.focus();
  }, [focusTarget]);

  const lastPage = pageOfPeriod(Math.max(0, periods.length - 1));
  const currentPage = Math.min(page, lastPage);
  const offset = currentPage * periodsPerPage;

  return (
    <fieldset className={styles.day}>
      <legend>{day}</legend>
      {lastPage > 0 ? (
        <SelectField
          label="Página de períodos"
          aria-label={`Página de períodos de ${day}`}
          fieldClassName={styles.pagination}
          value={currentPage}
          disabled={submitting}
          onChange={(event) =>
            onPageChange(weekday, Number(event.currentTarget.value))
          }
        >
          {Array.from({ length: lastPage + 1 }, (_, index) => (
            <option key={index} value={index}>
              {index * periodsPerPage + 1} a{" "}
              {Math.min((index + 1) * periodsPerPage, periods.length)} de{" "}
              {periods.length}
            </option>
          ))}
        </SelectField>
      ) : null}
      {periods.slice(offset, offset + periodsPerPage).map((period, index) => {
        const focusField =
          focusTarget?.id === period.id ? focusTarget.field : undefined;
        return (
          <SchedulePeriodRow
            key={period.id}
            period={period}
            day={day}
            number={offset + index + 1}
            writable={writable}
            submitting={submitting}
            focusField={focusField}
            focusRef={focusField ? focusRef : undefined}
            onEdit={onEdit}
            onRemove={onRemove}
          />
        );
      })}
      {writable ? (
        <Button
          aria-label={`Adicionar período em ${day}`}
          type="button"
          variant="secondary"
          disabled={submitting}
          onClick={() => onAdd(weekday, pageOfPeriod(periods.length))}
        >
          Adicionar período
        </Button>
      ) : periods.length === 0 ? (
        <p>Nenhum período.</p>
      ) : null}
    </fieldset>
  );
});

const SchedulePeriodRow = memo(function SchedulePeriodRow({
  period,
  day,
  number,
  writable,
  submitting,
  focusField,
  focusRef,
  onEdit,
  onRemove,
}: RowProps) {
  return (
    <div className={styles.period}>
      {timeFields.map(({ name, label }) => (
        <TextField
          key={name}
          ref={focusField === name ? focusRef : undefined}
          aria-label={`${label} do período ${number} de ${day}`}
          label={label}
          placeholder="HH:mm"
          maxLength={5}
          readOnly={!writable || submitting}
          value={period[name]}
          onChange={(event) => onEdit(period, name, event.currentTarget.value)}
        />
      ))}
      {writable ? (
        <Button
          aria-label={`Remover período ${number} de ${day}`}
          type="button"
          variant="secondary"
          disabled={submitting}
          onClick={() => onRemove(period)}
        >
          Remover
        </Button>
      ) : null}
    </div>
  );
});
