import { useState } from "react";
import { QueryClientProvider, type QueryClient } from "@tanstack/react-query";
import { I18nextProvider } from "react-i18next";
import { Router, type RouterProps } from "wouter";

import { createAppQueryClient } from "./app/queryClient";
import { AppRoutes } from "./app/routes";
import { i18n } from "./i18n";

interface AppProps {
  locationHook?: RouterProps["hook"];
  queryClient?: QueryClient;
}

export function App({ locationHook, queryClient }: AppProps) {
  const [activeQueryClient] = useState(
    () => queryClient ?? createAppQueryClient(),
  );

  return (
    <I18nextProvider i18n={i18n}>
      <QueryClientProvider client={activeQueryClient}>
        <Router {...(locationHook === undefined ? {} : { hook: locationHook })}>
          <AppRoutes />
        </Router>
      </QueryClientProvider>
    </I18nextProvider>
  );
}
