import React from 'react';
import { SceneApp, useSceneApp } from '@grafana/scenes';
import { homePage } from '../../pages/Home/homePage';

function getSceneApp() {
  return new SceneApp({
    pages: [homePage],
    urlSyncOptions: {
      updateUrlOnInit: true,
      createBrowserHistorySteps: true,
    },
  });
}

function App() {
  const scene = useSceneApp(getSceneApp);
  return <scene.Component model={scene} />;
}

export default App;
