import { act, renderHook } from "@testing-library/react";
import { StrictMode } from "react";
import { expect, test, vi } from "vitest";
import { useMutationRequest } from "./use-mutation-request";

test("bloqueia início duplicado e libera uma nova tentativa após concluir", () => {
  const activity = vi.fn();
  const { result } = renderHook(() => useMutationRequest(activity), {
    wrapper: StrictMode,
  });
  let request!: NonNullable<ReturnType<typeof result.current.begin>>;
  act(() => {
    request = result.current.begin()!;
    expect(result.current.begin()).toBeNull();
  });
  expect(result.current.submitting).toBe(true);
  expect(activity.mock.calls).toEqual([[true]]);
  act(() => request.finish());
  expect(result.current.submitting).toBe(false);
  act(() => {
    expect(result.current.begin()).not.toBeNull();
  });
  expect(activity.mock.calls).toEqual([[true], [false], [true]]);
});

test("desmontar cancela e libera a atividade uma vez, mesmo antes do finally tardio", () => {
  const activity = vi.fn();
  const { result, unmount } = renderHook(() => useMutationRequest(activity));
  let request!: NonNullable<ReturnType<typeof result.current.begin>>;
  act(() => {
    request = result.current.begin()!;
  });
  unmount();
  expect(request.signal.aborted).toBe(true);
  expect(activity.mock.calls).toEqual([[true], [false]]);
  request.finish();
  expect(activity.mock.calls).toEqual([[true], [false]]);
});
