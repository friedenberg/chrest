import { nodeResolve } from '@rollup/plugin-node-resolve';

export default {
  input: 'src/main.js',
  output: {
    format: 'cjs',
  },
  plugins: [nodeResolve()]
};
