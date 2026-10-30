import { expect, test } from "vitest";
import { classifyProfessionalOperation } from "./professional-operation-result";
import type { Professional } from "./professionals-api";

test("trata resposta 200 sem contrato e falha de infraestrutura como resultado incerto", () => {
  expect(
    classifyProfessionalOperation(
      { state: "success", status: 200, data: {} as Professional },
      200,
    ).state,
  ).toBe("uncertain");
  expect(
    classifyProfessionalOperation(
      {
        state: "error",
        status: 500,
        error: { code: "read_only", message: "privado", request_id: "id" },
      },
      200,
    ).state,
  ).toBe("uncertain");
  expect(
    classifyProfessionalOperation(
      {
        state: "error",
        status: 403,
        error: { code: "read_only", message: "privado", request_id: "id" },
      },
      200,
    ).state,
  ).toBe("read_only");
});
