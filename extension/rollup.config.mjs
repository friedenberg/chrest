import { nodeResolve } from '@rollup/plugin-node-resolve';
import consts from 'rollup-plugin-consts';

export default (commandLineArgs) => ({
  input: 'src/main.js',
  output: {
    format: 'cjs',
  },
  plugins: [
    nodeResolve(),
    consts({
      browserType: commandLineArgs.browserType,
    })
  ]
});
