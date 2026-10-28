import { useState } from "react";
import { fireEvent, render, screen } from "@testing-library/react";
import { expect, test, vi } from "vitest";

import { BrazilianPhoneField } from "./brazilian-phone-field";

test("applies Brazilian landline and mobile masks and limits the input length", () => {
  const onValueChange = vi.fn();
  const { rerender } = render(
    <BrazilianPhoneField
      id="contact_phone"
      label="Telefone de contato"
      onValueChange={onValueChange}
      value=""
    />,
  );

  const phoneField = screen.getByLabelText("Telefone de contato");
  expect(phoneField).toHaveAttribute("type", "tel");
  expect(phoneField).toHaveAttribute("maxLength", "15");
  expect(phoneField).toHaveAttribute("placeholder", "(00) 00000-0000");

  fireEvent.change(phoneField, { target: { value: "1133334444" } });
  expect(onValueChange).toHaveBeenLastCalledWith("(11) 3333-4444");

  rerender(
    <BrazilianPhoneField
      id="contact_phone"
      label="Telefone de contato"
      onValueChange={onValueChange}
      value="(11) 3333-4444"
    />,
  );
  fireEvent.change(phoneField, { target: { value: "11987654321" } });
  expect(onValueChange).toHaveBeenLastCalledWith("(11) 98765-4321");
});

test("formats a persisted Brazilian number with its country code for display", () => {
  render(
    <BrazilianPhoneField
      id="contact_phone"
      label="Telefone de contato"
      onValueChange={vi.fn()}
      value="+5511999990000"
    />,
  );

  expect(screen.getByLabelText("Telefone de contato")).toHaveValue(
    "(11) 99999-0000",
  );
});

test("limits entries to eleven national digits and accepts a pasted country code", () => {
  const onValueChange = vi.fn();
  render(
    <BrazilianPhoneField
      id="contact_phone"
      label="Telefone de contato"
      onValueChange={onValueChange}
      value=""
    />,
  );

  const phoneField = screen.getByLabelText("Telefone de contato");
  fireEvent.change(phoneField, { target: { value: "119876543219999" } });
  expect(onValueChange).toHaveBeenLastCalledWith("(11) 98765-4321");

  fireEvent.paste(phoneField, {
    clipboardData: { getData: () => "+55 (11) 99999-0000" },
  });
  expect(onValueChange).toHaveBeenLastCalledWith("(11) 99999-0000");
});

test("preserves the cursor position when editing a digit in the middle", () => {
  render(<ControlledPhoneField initialValue="(11) 98765-432" />);

  const phoneField = screen.getByLabelText(
    "Telefone de contato",
  ) as HTMLInputElement;
  fireEvent.change(phoneField, {
    target: {
      value: "11987695432",
      selectionStart: 7,
      selectionEnd: 7,
    },
  });

  expect(phoneField).toHaveValue("(11) 98769-5432");
  expect(phoneField.selectionStart).toBe(10);
});

function ControlledPhoneField({ initialValue }: { initialValue: string }) {
  const [value, setValue] = useState(initialValue);

  return (
    <BrazilianPhoneField
      id="contact_phone"
      label="Telefone de contato"
      onValueChange={setValue}
      value={value}
    />
  );
}
