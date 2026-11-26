"use client";

import {
  useEffect,
  useEffectEvent,
  useRef,
  useState,
  type FormEvent,
} from "react";
import { Button, Feedback, ModalActions, TextField } from "@/components/ui";
import { useMutationRequest } from "@/lib/api/use-mutation-request/use-mutation-request";
import {
  WEEKDAYS,
  compareWorkPeriods,
  parseWeeklySchedule,
  sameWorkPeriods,
  validateWorkPeriods,
} from "@/lib/professionals/weekly-schedule";
import {
  getProfessionalWeeklySchedule,
  putProfessionalWeeklySchedule,
  type WeeklySchedule,
  type WorkPeriod,
} from "@/lib/professionals/weekly-schedule-api";
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
  | { status: "ready"; week: WeeklySchedule };
type DraftPeriod = WorkPeriod & { id: number };
type Notice = { variant: "success" | "error"; title: string; message: string };

const saveError: Notice = {
  variant: "error",
  title: "Não foi possível salvar a jornada",
  message: "Confira o rascunho e use Salvar para tentar novamente.",
};

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
  const [draft, setDraft] = useState<DraftPeriod[]>([]);
  const [notice, setNotice] = useState<Notice | null>(null);
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
        setDraft(week.periods.map((period, id) => ({ ...period, id })));
        nextId.current = week.periods.length;
        setView({ status: "ready", week });
      })
      .catch(() => {
        if (!controller.signal.aborted) setView({ status: "error" });
      });
    return () => controller.abort();
  }, [professionalId, reload]);

  const writable = canWrite && view.status === "ready" && view.week.can_write;
  function edit(change: (previous: DraftPeriod[]) => DraftPeriod[]) {
    if (!writable || isPending()) return;
    setNotice(null);
    setDraft(change);
  }

  async function save(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (view.status !== "ready" || !writable || isPending()) return;
    const input = validateWorkPeriods(draft);
    if (!input.valid) {
      setNotice({
        variant: "error",
        title: "Confira os períodos",
        message: input.message,
      });
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
        setView({ status: "ready", week: { ...view.week, can_write: false } });
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
        setNotice(saveError);
        return;
      }
      setView({ status: "ready", week: confirmed });
      setDraft((periods) => [...periods].sort(compareWorkPeriods));
      setNotice({
        variant: "success",
        title: "Jornada salva",
        message: "Os horários foram atualizados.",
      });
    } catch {
      if (!request.signal.aborted) setNotice(saveError);
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
    <Button type="submit" disabled={!writable || submitting}>
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
        Fuso horário: <strong>{view.week.time_zone}</strong>
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
        const periods = draft.filter((period) => period.weekday === weekday);
        return (
          <fieldset className={styles.day} key={day}>
            <legend>{day}</legend>
            {periods.map((period, index) => {
              const label = `do período ${index + 1} de ${day}`;
              return (
                <div className={styles.period} key={period.id}>
                  <TextField
                    aria-label={`Início ${label}`}
                    label="Início"
                    placeholder="HH:mm"
                    maxLength={5}
                    readOnly={!writable || submitting}
                    value={period.start_time}
                    onChange={(event) => {
                      const start_time = event.currentTarget.value;
                      edit((current) =>
                        current.map((item) =>
                          item.id === period.id
                            ? { ...item, start_time }
                            : item,
                        ),
                      );
                    }}
                  />
                  <TextField
                    aria-label={`Fim ${label}`}
                    label="Fim"
                    placeholder="HH:mm"
                    maxLength={5}
                    readOnly={!writable || submitting}
                    value={period.end_time}
                    onChange={(event) => {
                      const end_time = event.currentTarget.value;
                      edit((current) =>
                        current.map((item) =>
                          item.id === period.id ? { ...item, end_time } : item,
                        ),
                      );
                    }}
                  />
                  {writable ? (
                    <Button
                      aria-label={`Remover período ${index + 1} de ${day}`}
                      type="button"
                      variant="secondary"
                      disabled={submitting}
                      onClick={() =>
                        edit((current) =>
                          current.filter((item) => item.id !== period.id),
                        )
                      }
                    >
                      Remover
                    </Button>
                  ) : null}
                </div>
              );
            })}
            {writable ? (
              <Button
                aria-label={`Adicionar período em ${day}`}
                type="button"
                variant="secondary"
                disabled={submitting}
                onClick={() => {
                  const id = nextId.current++;
                  edit((current) => [
                    ...current,
                    { id, weekday, start_time: "", end_time: "" },
                  ]);
                }}
              >
                Adicionar período
              </Button>
            ) : periods.length === 0 ? (
              <p>Nenhum período.</p>
            ) : null}
          </fieldset>
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
