#!/usr/bin/env bash
# Vendors the shadcn-svelte components of the SvelteKit app. It renders a
# SvelteKit project into a temporary directory, runs shadcn-svelte add there,
# and copies the components, hooks, utils.ts, package.json and pnpm-lock.yaml
# back into generator/_template/web.svelte. pnpm never runs inside _template.
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
generator_dir="$(cd "${script_dir}/.." && pwd)"
web_dir="${generator_dir}/_template/web.svelte"
version="1.7.0"
work_root="$(mktemp -d)"
trap 'rm -rf "${work_root}"' EXIT

components=(
  alert alert-dialog avatar badge button card chart dialog dropdown-menu empty
  field input input-group item label pagination select separator sheet sidebar
  skeleton table tabs textarea tooltip
)

# Any SvelteKit selection works: the vendored files, package.json and
# pnpm-lock.yaml are the same in every selection.
sed 's/"framework": "headless"/"framework": "svelte"/' \
  "${generator_dir}/testdata/configs/headless-workspaces-polar-postgres-resend.json" \
  > "${work_root}/config.json"

go -C "${generator_dir}" run ./cmd/render \
  -config "${work_root}/config.json" \
  -version v0.0.0-sync \
  -out "${work_root}/project"

project_web="${work_root}/project/web"
cd "${project_web}"

installed="$(node -p "require('./package.json').devDependencies['shadcn-svelte']")"
if [[ "${installed}" != "${version}" ]]; then
  printf 'web.svelte/package.json pins shadcn-svelte %s, the script expects %s.\n' "${installed}" "${version}" >&2
  exit 1
fi

pnpm install
rm -rf src/lib/components/ui src/lib/hooks src/lib/utils.ts
pnpm exec shadcn-svelte add utils "${components[@]}" --yes --overwrite

# shadcn-svelte adds registry dependencies with a caret range. Pin each one to
# the version pnpm installed, so package.json only carries exact versions.
node --input-type=module <<'JS'
import fs from 'node:fs';

const manifest = JSON.parse(fs.readFileSync('package.json', 'utf8'));
for (const field of ['dependencies', 'devDependencies']) {
	for (const [name, range] of Object.entries(manifest[field] ?? {})) {
		if (/^\d/.test(range)) continue;
		const installed = JSON.parse(fs.readFileSync(`node_modules/${name}/package.json`, 'utf8'));
		manifest[field][name] = installed.version;
	}
}
fs.writeFileSync('package.json', JSON.stringify(manifest, null, '\t') + '\n');
JS
pnpm install

rm -rf "${web_dir}/src/lib/components/ui" "${web_dir}/src/lib/hooks"
mkdir -p "${web_dir}/src/lib/components"
cp -a src/lib/components/ui "${web_dir}/src/lib/components/ui"
cp -a src/lib/hooks "${web_dir}/src/lib/hooks"
cp src/lib/utils.ts "${web_dir}/src/lib/utils.ts"
cp package.json pnpm-lock.yaml "${web_dir}/"

printf 'Installed shadcn-svelte %s components in generator/_template/web.svelte.\n' "${version}"
