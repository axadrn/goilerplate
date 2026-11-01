#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
generator_dir="$(cd "${script_dir}/.." && pwd)"
template_dir="${generator_dir}/_template"
generator_target_dir="${template_dir}/internal/ui/components"
version="${SHADCN_TEMPL_VERSION:-v2.0.0}"
work_root="$(mktemp -d)"
trap 'rm -rf "${work_root}"' EXIT

generator_components=(
  accordion alert aspect-ratio avatar badge breadcrumb button calendar card chart
  collapsible dialog dropdown-menu empty field icon input input-group item label
  pagination popover progress select separator sheet sidebar skeleton switch
  table tabs textarea toast tooltip
)

sync_base_assets() {
  local work_dir="${work_root}/base"

  mkdir -p "${work_dir}"
  printf 'module example.com/goilerplate-shadcn-base\n\ngo 1.26.0\n' > "${work_dir}/go.mod"
  go run "github.com/axadrn/shadcn-templ/v2/cmd/shadcn-templ@${version}" \
    init --cwd "${work_dir}" --css assets/css/globals.css --base-color stone --silent

  cp "${work_dir}/assets/css/shadcn-tailwind.css" "${template_dir}/assets.htmx.datastar/css/shadcn-tailwind.css"
  cp "${work_dir}/assets/css/tw-animate.css" "${template_dir}/assets.htmx.datastar/css/tw-animate.css"
}

install_components() {
  local work_dir="$1"
  local module_path="$2"
  local css_source="$3"
  local components_alias="$4"
  local utils_alias="$5"
  shift 5

  mkdir -p "${work_dir}/assets/css"
  printf 'module %s\n\ngo 1.26.0\n' "${module_path}" > "${work_dir}/go.mod"
  cp "${css_source}" "${work_dir}/assets/css/globals.css"
  printf '%s\n' \
    '{' \
    '  "scripts": {' \
    '    "dir": "assets/js",' \
    '    "path": "/assets/js"' \
    '  },' \
    '  "$schema": "https://shadcn-templ.com/schema/components.json",' \
    '  "style": "base-nova",' \
    '  "tailwind": {' \
    '    "css": "assets/css/globals.css",' \
    '    "baseColor": "stone",' \
    '    "cssVariables": true' \
    '  },' \
    '  "rtl": false,' \
    '  "iconLibrary": "lucide",' \
    '  "menuColor": "default",' \
    '  "menuAccent": "subtle",' \
    '  "aliases": {' \
    "    \"components\": \"${components_alias}\"," \
    "    \"utils\": \"${utils_alias}\"" \
    '  }' \
    '}' > "${work_dir}/components.json"

  go run "github.com/axadrn/shadcn-templ/v2/cmd/shadcn-templ@${version}" \
    add utils "$@" --cwd "${work_dir}" --overwrite --silent
  go run "github.com/axadrn/shadcn-templ/v2/cmd/shadcn-templ@${version}" \
    bundle --cwd "${work_dir}" --silent
}

# copy_bundle replaces the bundled component JavaScript in a target assets/js.
# The file name carries a content hash, so stale bundles are removed first.
copy_bundle() {
  local work_dir="$1"
  local target_dir="$2"

  rm -f "${target_dir}"/shadcn-templ-*.js
  cp "${work_dir}"/assets/js/shadcn-templ-*.js "${target_dir}/"
}

sync_base_assets

generator_work_dir="${work_root}/generator"
generator_module_path="example.com/goilerplate-shadcn-sync"
install_components \
  "${generator_work_dir}" \
  "${generator_module_path}" \
  "${template_dir}/assets.htmx.datastar/css/globals.htmx.css" \
  "${generator_module_path}/internal/ui/components" \
  "${generator_module_path}/internal/ui/utils" \
  "${generator_components[@]}"

while IFS= read -r file; do
  sed -i "s#${generator_module_path}#{{.ModulePath}}#g" "${file}"
done < <(rg -l "${generator_module_path}" "${generator_work_dir}/internal/ui/components" || true)

rm -rf "${generator_target_dir}"
mkdir -p "${generator_target_dir}"
cp -a "${generator_work_dir}/internal/ui/components/." "${generator_target_dir}/"
cp "${generator_work_dir}/internal/ui/utils/shadcn-templ.go" "${template_dir}/internal/ui/utils.htmx.datastar/shadcn-templ.go"
copy_bundle "${generator_work_dir}" "${template_dir}/assets.htmx.datastar/js"

printf 'Installed shadcn-templ %s components in the generator from the live registry.\n' "${version}"
