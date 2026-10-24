"use client";

import { useState } from "react";

import { CreateProfessionalForm } from "./create-professional-form";
import { ProfessionalsListPanel } from "./professionals-list-panel";
import styles from "./professionals-workspace.module.css";

export function ProfessionalsWorkspace() {
  const [listVersion, setListVersion] = useState(0);

  return (
    <div className={styles.workspace}>
      <CreateProfessionalForm
        onCreated={() => setListVersion((value) => value + 1)}
      />
      <ProfessionalsListPanel key={listVersion} />
    </div>
  );
}
