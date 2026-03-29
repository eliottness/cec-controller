#!/usr/bin/env bash
# use-prebuilt.sh — goreleaser gobinary wrapper.
# Copies a pre-built binary to wherever goreleaser wants it instead of
# compiling from source. Used so that native per-arch builds from
# separate CI runners can be packaged by a single goreleaser invocation.
#
# goreleaser calls: <gobinary> build [-v] [-trimpath] -o <path> [-ldflags ...] <main>
set -e

OUTPUT=""
args=("$@")
i=0
while [[ $i -lt ${#args[@]} ]]; do
  if [[ "${args[$i]}" == "-o" ]]; then
    OUTPUT="${args[$((i+1))]}"
    break
  fi
  (( i++ )) || true
done

if [[ -z "$OUTPUT" ]]; then
  echo "use-prebuilt: -o flag not found in args: $*" >&2
  exit 1
fi

if [[ "$OUTPUT" == *"arm64"* ]]; then
  ARCH="arm64"
else
  ARCH="amd64"
fi

SRC="prebuilt/$ARCH/cec-controller"
if [[ ! -f "$SRC" ]]; then
  echo "use-prebuilt: pre-built binary not found: $SRC" >&2
  exit 1
fi

mkdir -p "$(dirname "$OUTPUT")"
cp "$SRC" "$OUTPUT"
chmod +x "$OUTPUT"
echo "use-prebuilt: $SRC -> $OUTPUT"
