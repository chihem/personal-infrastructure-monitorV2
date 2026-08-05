import { render, type RenderOptions } from "@testing-library/react";
import type { ReactNode } from "react";
import { I18nextProvider } from "react-i18next";

import { i18n } from "../i18n";

export function renderWithI18n(
  children: ReactNode,
  options?: Omit<RenderOptions, "wrapper">,
) {
  return render(
    <I18nextProvider i18n={i18n}>{children}</I18nextProvider>,
    options,
  );
}
