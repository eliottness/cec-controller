#!/usr/bin/env bash
# use-prebuilt.sh — goreleaser build `tool` wrapper (builds.tool in .goreleaser.yml).
# Copies a pre-built binary to wherever goreleaser wants it instead of
# compiling from source. Used so that native per-(libcec major, arch) builds
# from separate CI containers can be packaged by a single goreleaser
# invocation. (goreleaser's native "prebuilt" builder is Pro-only.)
#
# goreleaser calls: <tool> build [-v] [-trimpath] -o <path> [-ldflags ...] <main>
# The build id (e.g. "libcec6-amd64") is embedded in <path> as
# dist/<id>_<target>/..., which we parse to pick the right pre-built binary.
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

# libcec variant (which runtime ABI the binary links): comes from the build id.
case "$OUTPUT" in
  *libcec6*) VARIANT="libcec6" ;;
  *libcec7*) VARIANT="libcec7" ;;
  *fedora*)  VARIANT="fedora" ;;
  *)
    echo "use-prebuilt: cannot determine libcec variant from output path: $OUTPUT" >&2
    exit 1
    ;;
esac

if [[ "$OUTPUT" == *"arm64"* ]]; then
  ARCH="arm64"
else
  ARCH="amd64"
fi

SRC="prebuilt/$VARIANT/$ARCH/cec-controller"
if [[ ! -f "$SRC" ]]; then
  echo "use-prebuilt: pre-built binary not found: $SRC" >&2
  exit 1
fi

mkdir -p "$(dirname "$OUTPUT")"
cp "$SRC" "$OUTPUT"
chmod +x "$OUTPUT"
echo "use-prebuilt: $SRC -> $OUTPUT"
