#!/usr/bin/env bash
set -euo pipefail

repository_root="$(git rev-parse --show-toplevel)"
cd "${repository_root}"

mapfile -d '' -t staged_files < <(git diff --cached --name-only -z --diff-filter=ACMR)

go_files=()
python_files=()
python_git_files=()
frontend_files=()
frontend_git_files=()
for file in "${staged_files[@]}"; do
	case "${file}" in
	*.go)
		go_files+=("${file}")
		;;
	tests/integration/*.py)
		python_files+=("${file#tests/integration/}")
		python_git_files+=("${file}")
		;;
	frontend/*.js|frontend/*.jsx|frontend/*.ts|frontend/*.tsx|frontend/*.cjs|frontend/*.mjs|frontend/*.json|frontend/*.css|frontend/*.scss|frontend/*.md|frontend/*.yaml|frontend/*.yml)
		frontend_files+=("${file#frontend/}")
		frontend_git_files+=("${file}")
		;;
	esac
done

format_files=("${go_files[@]}" "${python_git_files[@]}" "${frontend_git_files[@]}")
for file in "${format_files[@]}"; do
	if [[ -n "$(git diff --name-only -- "${file}")" ]]; then
		echo "Cannot format partially staged file: ${file}. Stage or discard its remaining changes first." >&2
		exit 1
	fi
done

if (( ${#go_files[@]} )); then
	gofmt -s -w "${go_files[@]}"
	git add -- "${go_files[@]}"
fi

if (( ${#python_files[@]} )); then
	(
		cd tests/integration
		uv run autoflake --in-place --remove-all-unused-imports --remove-unused-variables "${python_files[@]}"
		uv run isort "${python_files[@]}"
		uv run black "${python_files[@]}"
	)
	git add -- "${python_git_files[@]}"
fi

if (( ${#frontend_files[@]} )); then
	prettier="${repository_root}/frontend/node_modules/.bin/prettier"
	if [[ ! -x "${prettier}" ]]; then
		echo "Frontend dependencies are required to format frontend source changes." >&2
		exit 1
	fi
	(
		cd frontend
		"${prettier}" --write "${frontend_files[@]}"
	)
	git add -- "${frontend_git_files[@]}"
fi
