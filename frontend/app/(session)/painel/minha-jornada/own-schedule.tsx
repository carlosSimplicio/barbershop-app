"use client";

import { WeeklyScheduleEditor } from "@/components/professionals/weekly-schedule-editor/weekly-schedule-editor";
import { useAuthSession } from "@/components/session/auth-provider/auth-provider";

export function OwnSchedule() {
  const { state } = useAuthSession();
  if (state.status !== "authenticated") return null;

  return (
    <WeeklyScheduleEditor
      professionalId={state.principal.user_id}
      sessionKey={state.principal.establishment_id}
    />
  );
}
