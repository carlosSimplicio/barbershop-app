import { render, screen, within } from "@testing-library/react";
import { expect, test } from "vitest";

import { RecordList } from "./record-list";

type Entry = { id: string; name: string };

test("apresenta registros como itens de uma lista nomeada", () => {
  const entries: Entry[] = [
    { id: "ana", name: "Ana Lima" },
    { id: "bia", name: "Beatriz Souza" },
  ];

  render(
    <RecordList
      emptyMessage="Nenhum registro encontrado."
      getKey={(entry) => entry.id}
      items={entries}
      label="Profissionais cadastrados"
      renderItem={(entry) => <strong>{entry.name}</strong>}
    />,
  );

  const list = screen.getByRole("list", { name: "Profissionais cadastrados" });
  const items = within(list).getAllByRole("listitem");

  expect(items).toHaveLength(2);
  expect(items[0]).toHaveTextContent("Ana Lima");
  expect(items[1]).toHaveTextContent("Beatriz Souza");
  expect(screen.queryByRole("status")).not.toBeInTheDocument();
});

test("anuncia o estado vazio e não cria itens fictícios", () => {
  render(
    <RecordList<Entry>
      emptyMessage="Ainda não há profissionais cadastrados."
      getKey={(entry) => entry.id}
      items={[]}
      label="Profissionais cadastrados"
      renderItem={(entry) => entry.name}
    />,
  );

  expect(screen.getByRole("status")).toHaveTextContent(
    "Ainda não há profissionais cadastrados.",
  );
  expect(screen.queryByRole("listitem")).not.toBeInTheDocument();
  expect(screen.queryByRole("list")).not.toBeInTheDocument();
});
