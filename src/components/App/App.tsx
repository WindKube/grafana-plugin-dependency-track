import React from 'react';
import { config, getDataSourceSrv } from '@grafana/runtime';
import { SceneApp, useSceneApp } from '@grafana/scenes';
import { EmptyState, LinkButton } from '@grafana/ui';
import { DS_PLUGIN_ID } from '../../scenes/common';
import { overviewPage } from '../../pages/overview';
import { projectsPage } from '../../pages/projects';
import { findingsPage } from '../../pages/findings';
import { violationsPage } from '../../pages/violations';
import { freshnessPage } from '../../pages/freshness';

function getSceneApp() {
  return new SceneApp({
    pages: [overviewPage, projectsPage, findingsPage, violationsPage, freshnessPage],
    urlSyncOptions: {
      updateUrlOnInit: true,
      createBrowserHistorySteps: true,
    },
  });
}

function Scenes() {
  const scene = useSceneApp(getSceneApp);
  return <scene.Component model={scene} />;
}

function App() {
  if (getDataSourceSrv().getList({ pluginId: DS_PLUGIN_ID }).length === 0) {
    return (
      <EmptyState
        variant="call-to-action"
        message="Connect Grafana to Dependency-Track"
        button={
          <LinkButton href={`${config.appSubUrl}/connections/datasources/new`} icon="plus">
            Add a Dependency-Track data source
          </LinkButton>
        }
      >
        The app needs a Dependency-Track data source with an API key that can view the portfolio, vulnerabilities and
        policy violations.
      </EmptyState>
    );
  }
  return <Scenes />;
}

export default App;
