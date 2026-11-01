"use client";

import { useCallback, useEffect, useRef, useState } from "react";

type MutationRequest = {
  signal: AbortSignal;
  finish: () => void;
};
type ActiveRequest = MutationRequest & { abort: () => void };

export function useMutationRequest(onBusyChange?: (pending: boolean) => void) {
  const [submitting, setSubmitting] = useState(false);
  const active = useRef<ActiveRequest | null>(null);
  useEffect(
    () => () => {
      active.current?.abort();
      active.current?.finish();
    },
    [],
  );
  const isPending = useCallback(() => active.current !== null, []);
  const begin = useCallback((): MutationRequest | null => {
    if (active.current) return null;
    const controller = new AbortController();
    let released = false;
    const request: ActiveRequest = {
      signal: controller.signal,
      abort: () => controller.abort(),
      finish: () => {
        // Cleanup and finally can both finish the same request. Release its
        // contribution once, even when the transport never settles after abort.
        if (released) return;
        released = true;
        active.current = null;
        onBusyChange?.(false);
        if (!controller.signal.aborted) setSubmitting(false);
      },
    };
    active.current = request;
    setSubmitting(true);
    onBusyChange?.(true);
    return request;
  }, [onBusyChange]);
  return { submitting, begin, isPending };
}
