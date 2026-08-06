import { Route, Switch } from "wouter";

import { AppShell } from "../components/AppShell";
import { CPUPage } from "../features/cpu/CPUPage";
import { NotFoundPage } from "../pages/NotFoundPage";
import { OverviewPage } from "../pages/OverviewPage";
import { PlaceholderPage } from "../pages/PlaceholderPage";

export function AppRoutes() {
  return (
    <AppShell>
      <Switch>
        <Route path="/" component={OverviewPage} />
        <Route path="/cpu" component={CPUPage} />
        <Route path="/memory">
          <PlaceholderPage page="memory" />
        </Route>
        <Route path="/filesystems">
          <PlaceholderPage page="filesystems" />
        </Route>
        <Route path="/docker">
          <PlaceholderPage page="docker" />
        </Route>
        <Route path="/events">
          <PlaceholderPage page="events" />
        </Route>
        <Route path="/audit">
          <PlaceholderPage page="audit" />
        </Route>
        <Route path="/backups">
          <PlaceholderPage page="backups" />
        </Route>
        <Route component={NotFoundPage} />
      </Switch>
    </AppShell>
  );
}
