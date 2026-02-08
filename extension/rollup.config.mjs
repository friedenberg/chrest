import { nodeResolve } from '@rollup/plugin-node-resolve';
import consts from 'rollup-plugin-consts';

export default (commandLineArgs) => {
  const plugins = [
    nodeResolve(),
    consts({
      browserType: commandLineArgs.browserType,
    })
  ];

  return [
    {
      input: 'src/main.js',
      output: {
        file: `dist-${commandLineArgs.browserType}/main.js`,
        format: 'cjs',
      },
      plugins,
    },
    {
      input: 'src/options.js',
      output: {
        file: `dist-${commandLineArgs.browserType}/options.js`,
        format: 'iife',
      },
      plugins,
    },
  ];
};
