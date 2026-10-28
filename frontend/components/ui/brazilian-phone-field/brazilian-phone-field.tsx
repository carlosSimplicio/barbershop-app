"use client";

import { useImperativeHandle, useLayoutEffect, useRef, useState } from "react";
import type { ClipboardEvent, ChangeEvent } from "react";

import { TextField } from "../text-field/text-field";
import type { TextFieldProps } from "../text-field/text-field";

export type BrazilianPhoneFieldProps = Omit<
  TextFieldProps,
  "inputMode" | "maxLength" | "onChange" | "onPaste" | "type" | "value"
> & {
  onValueChange: (value: string) => void;
  value: string;
};

const MAX_MASKED_PHONE_LENGTH = 15;

export function BrazilianPhoneField({
  onValueChange,
  value,
  ...props
}: BrazilianPhoneFieldProps) {
  const inputRef = useRef<HTMLInputElement | null>(null);
  const pendingCaretDigitCount = useRef<number | null>(null);
  const [caretRevision, setCaretRevision] = useState(0);
  const { ref: forwardedRef, ...fieldProps } = props;

  useImperativeHandle(
    forwardedRef,
    () => inputRef.current as HTMLInputElement,
    [],
  );

  useLayoutEffect(() => {
    const digitCount = pendingCaretDigitCount.current;
    const input = inputRef.current;
    if (digitCount === null || !input) return;

    const caretPosition = getCaretPosition(input.value, digitCount);
    input.setSelectionRange(caretPosition, caretPosition);
    pendingCaretDigitCount.current = null;
  }, [caretRevision, value]);

  function handleChange(event: ChangeEvent<HTMLInputElement>) {
    const input = event.currentTarget;
    const caretPosition = input.selectionStart ?? input.value.length;
    pendingCaretDigitCount.current = countDigits(
      input.value.slice(0, caretPosition),
    );
    setCaretRevision((revision) => revision + 1);
    onValueChange(formatBrazilianPhoneNumber(input.value));
  }

  function handlePaste(event: ClipboardEvent<HTMLInputElement>) {
    const pastedValue = event.clipboardData.getData("text");
    if (pastedValue.length <= MAX_MASKED_PHONE_LENGTH) return;

    event.preventDefault();
    const formattedValue = formatBrazilianPhoneNumber(pastedValue);
    pendingCaretDigitCount.current = countDigits(formattedValue);
    setCaretRevision((revision) => revision + 1);
    onValueChange(formattedValue);
  }

  return (
    <TextField
      autoComplete="tel"
      inputMode="tel"
      maxLength={MAX_MASKED_PHONE_LENGTH}
      onChange={handleChange}
      onPaste={handlePaste}
      placeholder={fieldProps.placeholder ?? "(00) 00000-0000"}
      ref={inputRef}
      type="tel"
      value={formatBrazilianPhoneNumber(value)}
      {...fieldProps}
    />
  );
}

function countDigits(value: string) {
  return value.replace(/\D/gu, "").length;
}

function getCaretPosition(value: string, digitCount: number) {
  if (digitCount === 0) return 0;

  let digitsSeen = 0;
  for (let index = 0; index < value.length; index += 1) {
    if (!/\d/u.test(value[index])) continue;
    digitsSeen += 1;
    if (digitsSeen === digitCount) return index + 1;
  }
  return value.length;
}

export function formatBrazilianPhoneNumber(value: string) {
  const trimmedValue = value.trim();
  let digits = value.replace(/\D/gu, "");

  if (trimmedValue.startsWith("+") && digits.startsWith("55")) {
    digits = digits.slice(2);
  } else if (
    (digits.length === 12 || digits.length === 13) &&
    digits.startsWith("55")
  ) {
    digits = digits.slice(2);
  }

  digits = digits.slice(0, 11);
  if (!digits) return "";
  if (digits.length <= 2) {
    return `(${digits}${digits.length === 2 ? ")" : ""}`;
  }

  const areaCode = digits.slice(0, 2);
  const subscriberNumber = digits.slice(2);
  const prefixLength = subscriberNumber.startsWith("9") ? 5 : 4;
  const formattedSubscriberNumber =
    subscriberNumber.length > prefixLength
      ? `${subscriberNumber.slice(0, prefixLength)}-${subscriberNumber.slice(prefixLength)}`
      : subscriberNumber;

  return `(${areaCode}) ${formattedSubscriberNumber}`;
}
