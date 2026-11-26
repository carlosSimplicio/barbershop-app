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
import { formatOfferingPrice } from "@/lib/catalog/money";
import type { Professional } from "@/lib/professionals/professionals-api";
import {
  listProfessionalServices,
  type ProfessionalServicesPage,
} from "@/lib/professionals/professional-services-api";
import { isProfessionalServicesPage } from "@/lib/professionals/professional-services-result";

type Props = {
  professional: Pick<Professional, "id" | "name" | "active">;
  canWrite: boolean;
  onClose: () => void;
  onUnauthorized: () => void;
  returnFocusRef?: RefObject<HTMLElement | null>;
};
type View =
  | { status: "loading" | "error" | "unauthorized" }
  | { status: "missing" }
  | { status: "success"; page: ProfessionalServicesPage };

export function ProfessionalServicesPanel(props: Props) {
  return <ProfessionalServicesRequest key={props.professional.id} {...props} />;
}

function ProfessionalServicesRequest({
  professional,
  canWrite,
  onClose,
  onUnauthorized,
  returnFocusRef,
}: Props) {
  const [view, setView] = useState<View>({ status: "loading" });
  const [offset, setOffset] = useState(FIRST_PAGE_OFFSET);
  const [version, setVersion] = useState(0);
  const unauthorized = useRef(false);
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
          setView({ status: "missing" });
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
    setView({ status: "loading" });
    setVersion((value) => value + 1);
  }
  if (view.status === "unauthorized") return null;
  return (
    <Modal
      open
      title={`Serviços de ${view.status === "success" ? view.page.professional.name : professional.name}`}
      onClose={onClose}
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
      {view.status === "missing" ? (
        <Feedback variant="warning" title="Profissional não encontrado">
          O cadastro selecionado não está disponível.
        </Feedback>
      ) : (
        <RecordListPanel
          title="Serviços do estabelecimento"
          label="Lista de serviços do profissional"
          paginationLabel="Paginação de serviços do profissional"
          view={view}
          busy={false}
          onRefresh={refresh}
          onPageChange={(next) => {
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
        <Button variant="secondary" onClick={onClose}>
          Fechar
        </Button>
      </ModalActions>
    </Modal>
  );
}
