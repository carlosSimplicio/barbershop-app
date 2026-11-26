"use client";

import { useEffect, useRef, useState, type RefObject } from "react";
import {
  Button,
  Feedback,
  Modal,
  ModalActions,
  RecordListPanel,
  RecordSummary,
} from "@/components/ui";
import { DEFAULT_PAGE_SIZE, FIRST_PAGE_OFFSET } from "@/lib/api/pagination";
import { useMutationRequest } from "@/lib/api/use-mutation-request/use-mutation-request";
import { formatOfferingPrice } from "@/lib/catalog/money";
import type { Professional } from "@/lib/professionals/professionals-api";
import {
  listProfessionalServices,
  setProfessionalService,
  type ProfessionalServicesPage,
} from "@/lib/professionals/professional-services-api";
import {
  classifyProfessionalServiceOperation,
  isProfessionalServicesPage,
} from "@/lib/professionals/professional-services-result";

type Props = {
  professional: Pick<Professional, "id" | "name" | "active">;
  canWrite: boolean;
  onReadOnly: () => void;
  onBusyChange: (pending: boolean) => void;
  onClose: () => void;
  onUnauthorized: () => void;
  returnFocusRef?: RefObject<HTMLElement | null>;
};
type View =
  | { status: "loading" | "error" | "unauthorized" }
  | { status: "missing"; resource: "professional" | "association" }
  | { status: "success"; page: ProfessionalServicesPage };

type Attempt = { serviceId: string; associated: boolean };
type Notice = {
  variant: "success" | "warning" | "error";
  title: string;
  message: string;
  retry?: Attempt;
};

export function ProfessionalServicesPanel(props: Props) {
  return <ProfessionalServicesRequest key={props.professional.id} {...props} />;
}

function ProfessionalServicesRequest({
  professional,
  canWrite,
  onClose,
  onReadOnly,
  onBusyChange,
  onUnauthorized,
  returnFocusRef,
}: Props) {
  const [view, setView] = useState<View>({ status: "loading" });
  const [offset, setOffset] = useState(FIRST_PAGE_OFFSET);
  const [version, setVersion] = useState(0);
  const unauthorized = useRef(false);
  const [notice, setNotice] = useState<Notice | null>(null);
  const { submitting, begin, isPending } = useMutationRequest(onBusyChange);
  useEffect(() => {
    const controller = new AbortController();
    void listProfessionalServices(professional.id, {
      limit: DEFAULT_PAGE_SIZE,
      offset,
      signal: controller.signal,
    })
      .then((result) => {
        if (controller.signal.aborted || unauthorized.current) return;
        if (result.status === 401) {
          unauthorized.current = true;
          setView({ status: "unauthorized" });
          onUnauthorized();
          return;
        }
        if (result.status === 404) {
          setView({ status: "missing", resource: "professional" });
          return;
        }
        if (
          result.state === "success" &&
          result.status === 200 &&
          isProfessionalServicesPage(result.data) &&
          result.data.professional.id === professional.id &&
          result.data.offset === offset &&
          result.data.limit === DEFAULT_PAGE_SIZE
        ) {
          setView({ status: "success", page: result.data });
          return;
        }
        setView({ status: "error" });
      })
      .catch(() => {
        if (!controller.signal.aborted && !unauthorized.current)
          setView({ status: "error" });
      });
    return () => controller.abort();
  }, [professional.id, offset, version, onUnauthorized]);
  function refresh() {
    if (isPending()) return;
    setNotice(null);
    setView({ status: "loading" });
    setVersion((value) => value + 1);
  }
  function close() {
    if (!isPending()) onClose();
  }
  function showFailure(attempt: Attempt) {
    setNotice({
      variant: "error",
      title: "Não foi possível salvar o vínculo",
      message: "Você pode tentar novamente ou fechar esta janela.",
      retry: attempt,
    });
  }
  async function mutate(attempt: Attempt) {
    if (
      isPending() ||
      !canWrite ||
      view.status !== "success" ||
      !view.page.can_write ||
      !view.page.items.some((item) => item.id === attempt.serviceId)
    )
      return;
    const request = begin();
    if (!request) return;
    setNotice(null);
    try {
      const result = await setProfessionalService(
        professional.id,
        attempt.serviceId,
        { associated: attempt.associated },
        request.signal,
      );
      if (request.signal.aborted || unauthorized.current) return;
      const outcome = classifyProfessionalServiceOperation(
        result,
        attempt.serviceId,
        attempt.associated,
      );
      if (outcome.state === "unauthorized") {
        unauthorized.current = true;
        setView({ status: "unauthorized" });
        onUnauthorized();
        return;
      }
      if (outcome.state === "missing") {
        setView({ status: "missing", resource: "association" });
        return;
      }
      if (outcome.state === "read_only") {
        setView((current) =>
          current.status === "success"
            ? { ...current, page: { ...current.page, can_write: false } }
            : current,
        );
        onReadOnly();
        return;
      }
      if (outcome.state === "forbidden") {
        setNotice({
          variant: "warning",
          title: "Alteração não autorizada",
          message:
            "Não foi possível autorizar a solicitação. Você pode fechar esta janela.",
        });
        return;
      }
      if (outcome.state === "confirmed") {
        // Pages cannot change during a mutation. Keyed instances abort on a different
        // professional or identity, so only the current selection accepts this result.
        setView((current) =>
          current.status === "success" &&
          current.page.professional.id === professional.id &&
          current.page.offset === offset
            ? {
                ...current,
                page: {
                  ...current.page,
                  items: current.page.items.map((item) =>
                    item.id === outcome.record.id ? outcome.record : item,
                  ),
                },
              }
            : current,
        );
        setNotice({
          variant: "success",
          title: attempt.associated
            ? "Serviço associado"
            : "Associação removida",
          message: "A alteração foi salva.",
        });
        return;
      }
      showFailure(attempt);
    } catch {
      if (!request.signal.aborted && !unauthorized.current)
        showFailure(attempt);
    } finally {
      request.finish();
    }
  }
  if (view.status === "unauthorized") return null;
  return (
    <Modal
      open
      title={`Serviços de ${view.status === "success" ? view.page.professional.name : professional.name}`}
      onClose={close}
      closeDisabled={submitting}
      returnFocusRef={returnFocusRef}
    >
      <p>
        Vínculos com profissional ou serviço inativo não habilitam novos
        agendamentos.
      </p>
      {!canWrite || (view.status === "success" && !view.page.can_write) ? (
        <Feedback variant="warning" title="Modo somente leitura">
          Você pode consultar os serviços. Alterações ficam disponíveis com
          acesso ativo.
        </Feedback>
      ) : null}
      {submitting ? <p role="status">Salvando vínculo…</p> : null}
      {notice ? (
        <Feedback
          variant={notice.variant}
          title={notice.title}
          action={
            notice.retry ? (
              <ModalActions>
                <Button
                  disabled={
                    !canWrite ||
                    submitting ||
                    view.status !== "success" ||
                    !view.page.can_write
                  }
                  onClick={() => void mutate(notice.retry!)}
                >
                  Tentar novamente
                </Button>
              </ModalActions>
            ) : undefined
          }
        >
          {notice.message}
        </Feedback>
      ) : null}
      {view.status === "missing" ? (
        <Feedback
          variant="warning"
          title={
            view.resource === "association"
              ? "Profissional ou serviço não encontrado"
              : "Profissional não encontrado"
          }
        >
          O cadastro selecionado não está disponível.
        </Feedback>
      ) : (
        <RecordListPanel
          title="Serviços do estabelecimento"
          label="Lista de serviços do profissional"
          paginationLabel="Paginação de serviços do profissional"
          view={view}
          busy={submitting}
          onRefresh={refresh}
          onPageChange={(next) => {
            if (isPending()) return;
            setNotice(null);
            setView({ status: "loading" });
            setOffset(next);
          }}
          messages={{
            loading: "Carregando serviços…",
            empty: "Nenhum serviço cadastrado.",
            errorTitle: "Não foi possível consultar os serviços",
            errorDescription: "Verifique sua conexão e tente novamente.",
          }}
          getKey={(item) => item.id}
          renderItem={(item) => (
            <RecordSummary
              title={item.name}
              actions={
                <ModalActions>
                  <Button
                    aria-label={`${item.associated ? "Remover" : "Associar"} ${item.name}`}
                    disabled={
                      !canWrite ||
                      submitting ||
                      view.status !== "success" ||
                      !view.page.can_write
                    }
                    onClick={() =>
                      void mutate({
                        serviceId: item.id,
                        associated: !item.associated,
                      })
                    }
                  >
                    {item.associated ? "Remover" : "Associar"}
                  </Button>
                </ModalActions>
              }
              status={{
                label: item.associated ? "Associado" : "Não associado",
                tone: item.associated ? "success" : "neutral",
              }}
            >
              <span>
                {formatOfferingPrice(item.price_cents)} ·{" "}
                {item.duration_minutes} minutos
              </span>
              <span>{item.active ? "Ativo" : "Inativo"}</span>
            </RecordSummary>
          )}
        />
      )}
      <ModalActions>
        <Button variant="secondary" onClick={close} disabled={submitting}>
          Fechar
        </Button>
      </ModalActions>
    </Modal>
  );
}
