import { EmbeddedScene, PanelBuilders, SceneFlexItem, SceneFlexLayout } from '@grafana/scenes';

export function homeScene() {
  return new EmbeddedScene({
    body: new SceneFlexLayout({
      children: [
        new SceneFlexItem({
          minHeight: 200,
          body: PanelBuilders.text()
            .setTitle('Getting started')
            .setOption(
              'content',
              'Add a **Dependency-Track** data source under *Connections → Data sources* to start exploring your portfolio.'
            )
            .build(),
        }),
      ],
    }),
  });
}
