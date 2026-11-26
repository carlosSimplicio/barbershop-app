"use client";

import {
  useCallback,
  useEffect,
  useEffectEvent,
  useRef,
  useState,
  type FormEvent,
} from "react";
import { Button, Feedback, ModalActions } from "@/components/ui";
import { useMutationRequest } from "@/lib/api/use-mutation-request/use-mutation-request";
import {
  WEEKDAYS,
  compareWorkPeriods,
  parseWeeklySchedule,
  sameWorkPeriods,
  validateWorkPeriods,
  type WorkPeriodTimeField,
} from "@/lib/professionals/weekly-schedule";
import {
  getProfessionalWeeklySchedule,
  putProfessionalWeeklySchedule,
  type WorkPeriod,
} from "@/lib/professionals/weekly-schedule-api";
import {
  WeeklyScheduleDay,
  pageOfPeriod,
  type DraftPeriod,
  type PeriodFocus,
} from "./weekly-schedule-day";
import styles from "./weekly-schedule-editor.module.css";

type Props = {
  professionalId: string;
  sessionKey?: string;
  canWrite?: boolean;
  onBusyChange?: (pending: boolean) => void;
  onReadOnly?: () => void;
  onUnauthorized?: () => void;
  onClose?: () => void;
};
type View =
  | { status: "loading" | "error" | "missing" | "unauthorized" }
  | {
      status: "ready";
      timeZone: string;
      canWrite: boolean;
      confirmedDraft: DraftPeriod[][];
    };
type Notice = { variant: "success" | "error"; title: string; message: string };

const saveError: Notice = {
  variant: "error",
  title: "Não foi possível salvar a jornada",
  message: "Confira o rascunho e use Salvar para tentar novamente.",
};

function toDraft(periods: readonly WorkPeriod[]) {
  const days: DraftPeriod[][] = WEEKDAYS.map(() => []);
  periods.forEach((period, id) => {
    days[period.weekday - 1].push({ ...period, id });
  });
  return days;
}

export function WeeklyScheduleEditor(props: Props) {
  return (
    <WeeklyScheduleRequest
      key={JSON.stringify([props.sessionKey, props.professionalId])}
      {...props}
    />
  );
}

function WeeklyScheduleRequest({
  professionalId,
  canWrite = true,
  onBusyChange,
  onReadOnly,
  onUnauthorized,
  onClose,
}: Props) {
  const [view, setView] = useState<View>({ status: "loading" });
  const [reload, setReload] = useState(0);
  const [draft, setDraft] = useState<DraftPeriod[][]>([]);
  const [pages, setPages] = useState(() => WEEKDAYS.map(() => 0));
  const [focusTarget, setFocusTarget] = useState<PeriodFocus | null>(null);
  const [notice, setNotice] = useState<Notice | null>(null);
  const [retrySave, setRetrySave] = useState(false);
  const nextId = useRef(0);
  const { begin, isPending, submitting } = useMutationRequest(onBusyChange);
  function invalidate() {
    setView({ status: "unauthorized" });
    setDraft([]);
    onUnauthorized?.();
  }
  const invalidateFromRead = useEffectEvent(invalidate);

  useEffect(() => {
    const controller = new AbortController();
    void getProfessionalWeeklySchedule(professionalId, controller.signal)
      .then((result) => {
        if (controller.signal.aborted) return;
        if (result.status === 401) {
          invalidateFromRead();
          return;
        }
        if (result.status === 404) {
          setView({ status: "missing" });
          return;
        }
        const week =
          result.state === "success" && result.status === 200
            ? parseWeeklySchedule(result.data, professionalId)
            : null;
        if (!week) {
          setView({ status: "error" });
          return;
        }
        const confirmedDraft = toDraft(week.periods);
        setDraft(confirmedDraft);
        nextId.current = week.periods.length;
        setView({
          status: "ready",
          timeZone: week.time_zone,
          canWrite: week.can_write,
          confirmedDraft,
        });
      })
      .catch(() => {
        if (!controller.signal.aborted) setView({ status: "error" });
      });
    return () => controller.abort();
  }, [professionalId, reload]);

  const writable = canWrite && view.status === "ready" && view.canWrite;
  const hasChanges =
    view.status === "ready" &&
    draft.some(
      (periods, index) =>
        periods !== view.confirmedDraft[index] &&
        !sameWorkPeriods(periods, view.confirmedDraft[index]),
    );
  const editDay = useCallback(
    (weekday: number, change: (previous: DraftPeriod[]) => DraftPeriod[]) => {
      if (!writable || isPending()) return false;
      setNotice(null);
      setFocusTarget(null);
      setDraft((days) =>
        days.map((periods, index) =>
          index === weekday - 1 ? change(periods) : periods,
        ),
      );
      return true;
    },
    [isPending, writable],
  );
  const selectPage = useCallback((weekday: number, page: number) => {
    setFocusTarget(null);
    setPages((previous) =>
      previous.map((current, index) =>
        index === weekday - 1 ? page : current,
      ),
    );
  }, []);
  const addPeriod = useCallback(
    (weekday: number, page: number) => {
      const id = nextId.current;
      if (
        !editDay(weekday, (periods) => [
          ...periods,
          { id, weekday, start_time: "", end_time: "" },
        ])
      )
        return;
      nextId.current++;
      selectPage(weekday, page);
      setFocusTarget({ id, weekday, field: "start_time" });
    },
    [editDay, selectPage],
  );
  const editPeriod = useCallback(
    (period: DraftPeriod, field: WorkPeriodTimeField, value: string) => {
      editDay(period.weekday, (periods) =>
        periods.map((current) =>
          current.id === period.id ? { ...current, [field]: value } : current,
        ),
      );
    },
    [editDay],
  );
  const removePeriod = useCallback(
    (period: DraftPeriod) => {
      editDay(period.weekday, (periods) =>
        periods.filter((current) => current.id !== period.id),
      );
    },
    [editDay],
  );

  async function save(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (
      view.status !== "ready" ||
      !writable ||
      isPending() ||
      (!hasChanges && !retrySave)
    )
      return;
    const input = validateWorkPeriods(draft.flat());
    if (!input.valid) {
      setNotice({
        variant: "error",
        title: "Confira os períodos",
        message: input.message,
      });
      if (input.target) {
        const { period, field } = input.target;
        const day = draft[period.weekday - 1];
        const index = day.findIndex((candidate) => candidate === period);
        if (index >= 0) {
          selectPage(period.weekday, pageOfPeriod(index));
          setFocusTarget({ id: day[index].id, weekday: period.weekday, field });
        }
      }
      return;
    }
    if (
      !retrySave &&
      sameWorkPeriods(input.periods, view.confirmedDraft.flat())
    ) {
      setDraft(view.confirmedDraft);
      setFocusTarget(null);
      setNotice(null);
      return;
    }

    const request = begin();
    if (!request) return;
    setNotice(null);
    try {
      const result = await putProfessionalWeeklySchedule(
        professionalId,
        { periods: input.periods },
        request.signal,
      );
      if (request.signal.aborted) return;
      if (result.status === 401) {
        invalidate();
        return;
      }
      if (
        result.state === "error" &&
        result.status === 403 &&
        result.error.code === "read_only"
      ) {
        setView({ ...view, canWrite: false });
        setNotice({
          variant: "error",
          title: "A jornada não foi salva",
          message: "As alterações permanecem no rascunho.",
        });
        onReadOnly?.();
        return;
      }
      const confirmed =
        result.state === "success" && result.status === 200
          ? parseWeeklySchedule(result.data, professionalId)
          : null;
      if (!confirmed || !sameWorkPeriods(confirmed.periods, input.periods)) {
        setRetrySave(true);
        setNotice(saveError);
        return;
      }
      const confirmedDraft = draft.map((periods) =>
        [...periods].sort(compareWorkPeriods),
      );
      setView({
        status: "ready",
        timeZone: confirmed.time_zone,
        canWrite: confirmed.can_write,
        confirmedDraft,
      });
      setDraft(confirmedDraft);
      setRetrySave(false);
      setNotice({
        variant: "success",
        title: "Jornada salva",
        message: "Os horários foram atualizados.",
      });
    } catch {
      if (!request.signal.aborted) {
        setRetrySave(true);
        setNotice(saveError);
      }
    } finally {
      request.finish();
    }
  }

  if (view.status === "unauthorized") return null;
  const cancel = onClose ? (
    <Button
      type="button"
      variant="secondary"
      disabled={submitting}
      onClick={() => {
        if (!isPending()) onClose();
      }}
    >
      Cancelar
    </Button>
  ) : null;

  if (view.status !== "ready") {
    return (
      <>
        {view.status === "loading" ? (
          <p role="status">Carregando jornada…</p>
        ) : (
          <Feedback
            variant="error"
            title={
              view.status === "missing"
                ? "Profissional não encontrado"
                : "Não foi possível consultar a jornada"
            }
            action={
              <Button
                type="button"
                onClick={() => {
                  setView({ status: "loading" });
                  setReload((value) => value + 1);
                }}
              >
                Tentar novamente
              </Button>
            }
          >
            Tente consultar a jornada novamente.
          </Feedback>
        )}
        {cancel ? <ModalActions>{cancel}</ModalActions> : null}
      </>
    );
  }

  const saveButton = (
    <Button
      type="submit"
      disabled={!writable || submitting || (!hasChanges && !retrySave)}
    >
      {submitting ? "Salvando…" : "Salvar"}
    </Button>
  );
  return (
    <form
      aria-label="Jornada semanal"
      className={styles.editor}
      noValidate
      onSubmit={save}
    >
      <p>
        Fuso horário: <strong>{view.timeZone}</strong>
      </p>
      {writable ? (
        <p>
          Use HH:mm. Para encerrar um período à meia-noite, informe 24:00 no
          fim.
        </p>
      ) : (
        <Feedback variant="warning" title="Modo somente leitura">
          Alterações ficam disponíveis com acesso ativo.
        </Feedback>
      )}
      {WEEKDAYS.map((day, dayIndex) => {
        const weekday = dayIndex + 1;
        return (
          <WeeklyScheduleDay
            key={day}
            day={day}
            weekday={weekday}
            periods={draft[dayIndex]}
            page={pages[dayIndex]}
            focusTarget={focusTarget?.weekday === weekday ? focusTarget : null}
            writable={writable}
            submitting={submitting}
            onPageChange={selectPage}
            onAdd={addPeriod}
            onEdit={editPeriod}
            onRemove={removePeriod}
          />
        );
      })}
      {notice ? (
        <Feedback title={notice.title} variant={notice.variant}>
          {notice.message}
        </Feedback>
      ) : null}
      {cancel ? (
        <ModalActions>
          {saveButton}
          {cancel}
        </ModalActions>
      ) : (
        <div className={styles.actions}>{saveButton}</div>
      )}
    </form>
  );
}
