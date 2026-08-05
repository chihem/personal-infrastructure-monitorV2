import { render } from "@testing-library/react";
import { memoryLocation } from "wouter/memory-location";

import { App } from "../App";

export function renderApp(path = "/") {
  const location = memoryLocation({ path });
  return {
    location,
    ...render(<App locationHook={location.hook} />),
  };
}
