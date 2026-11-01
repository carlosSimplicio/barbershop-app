import "@testing-library/jest-dom/vitest";
import { cleanup } from "@testing-library/react";
import { afterEach } from "vitest";

// jsdom does not implement the native dialog lifecycle or browser top layer.
// Tests exercise our controlled opening, dismissal and focus restoration;
// native focus containment and background inertness are checked in a browser.
HTMLDialogElement.prototype.showModal = function () {
  this.setAttribute("open", "");
};
HTMLDialogElement.prototype.close = function () {
  this.removeAttribute("open");
};

afterEach(cleanup);
