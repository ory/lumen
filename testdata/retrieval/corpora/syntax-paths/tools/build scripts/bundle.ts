import { build } from 'esbuild';

// Bundles the storefront entrypoint for production.
export async function bundleAssets(outdir: string): Promise<void> {
  await build({ entryPoints: ['src/main.ts'], bundle: true, minify: true, outdir });
}
