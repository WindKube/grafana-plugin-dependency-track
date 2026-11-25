import CopyWebpackPlugin from 'copy-webpack-plugin';
import type { Configuration } from 'webpack';
import { merge } from 'webpack-merge';
import grafanaConfig, { type Env } from './.config/webpack/webpack.config.ts';

const config = async (env: Env): Promise<Configuration> => {
  const baseConfig = await grafanaConfig(env);

  return merge(baseConfig, {
    plugins: [
      new CopyWebpackPlugin({
        patterns: [{ from: 'datasource/img', to: 'datasource/img' }],
      }),
    ],
  });
};

export default config;
