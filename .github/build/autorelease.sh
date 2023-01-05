#!/bin/bash
# GoWebDav AutoRelease script

# Make sure we can catch err code
set -e

# Check basic tools we need
for tool in {"openssl","sed"}
do
	command -v "${tool}" > "/dev/null" || { echo -e "${tool} not found."; exit 1; }
done

# Initial dependencies
echo -e "Initialing dependencies..."
go mod download || { echo -e "Failed to initial dependencies."; exit 1; }

# Prepare the build directory
build_dir="$PWD/build"
rm -rf "${build_dir}"
mkdir "${build_dir}" || { echo -e "Failed to create build dir."; exit 1; }

# Set build vars
readonly go_ldflags="-s -w -buildid="

#  os      arch      amd64  arm  mips       mips64     sse        snap
readonly platforms="\
android    arm64     -      -    -          -          -          aarch64
darwin     amd64     v1     -    -          -          -          -
darwin     amd64     v2     -    -          -          -          -
darwin     amd64     v3     -    -          -          -          -
darwin     amd64     v4     -    -          -          -          -
darwin     arm64     -      -    -          -          -          aarch64
dragonfly  amd64     v1     -    -          -          -          -
dragonfly  amd64     v2     -    -          -          -          -
dragonfly  amd64     v3     -    -          -          -          -
dragonfly  amd64     v4     -    -          -          -          -
freebsd    386       -      -    -          -          softfloat  386-softfloat
freebsd    386       -      -    -          -          sse2       386-sse2
freebsd    amd64     v1     -    -          -          -          -
freebsd    amd64     v2     -    -          -          -          -
freebsd    amd64     v3     -    -          -          -          -
freebsd    amd64     v4     -    -          -          -          -
freebsd    arm       -      7    -          -          -          armv7
freebsd    arm64     -      -    -          -          -          aarch64
linux      386       -      -    -          -          softfloat  386-softfloat
linux      386       -      -    -          -          sse2       386-sse2
linux      amd64     v1     -    -          -          -          -
linux      amd64     v2     -    -          -          -          -
linux      amd64     v3     -    -          -          -          -
linux      amd64     v4     -    -          -          -          -
linux      arm       -      5    -          -          -          armv5
linux      arm       -      6    -          -          -          armv6
linux      arm       -      7    -          -          -          armv7
linux      arm64     -      -    -          -          -          aarch64
linux      mips      -      -    hardfloat  -          -          -
linux      mips      -      -    softfloat  -          -          -
linux      mips64    -      -    -          hardfloat  -          -
linux      mips64    -      -    -          softfloat  -          -
linux      mips64le  -      -    -          hardfloat  -          -
linux      mips64le  -      -    -          softfloat  -          -
linux      mipsle    -      -    hardfloat  -          -          -
linux      mipsle    -      -    softfloat  -          -          -
linux      ppc64     -      -    -          -          -          -
linux      ppc64le   -      -    -          -          -          -
linux      riscv64   -      -    -          -          -          -
linux      s390x     -      -    -          -          -          -
openbsd    arm       -      7    -          -          -          armv7
openbsd    arm64     -      -    -          -          -          aarch64
openbsd    386       -      -    -          -          softfloat  386-softfloat
openbsd    386       -      -    -          -          sse2       386-sse2
openbsd    amd64     v1     -    -          -          -          -
openbsd    amd64     v2     -    -          -          -          -
openbsd    amd64     v3     -    -          -          -          -
openbsd    amd64     v4     -    -          -          -          -
windows    386       -      -    -          -          softfloat  x86_softfloat
windows    386       -      -    -          -          sse2       x86_sse2
windows    amd64     v1     -    -          -          -          x86_64_v1
windows    amd64     v2     -    -          -          -          x86_64_v2
windows    amd64     v3     -    -          -          -          x86_64_v3
windows    amd64     v4     -    -          -          -          x86_64_v4
windows    arm       -      6    -          -          -          armv6
windows    arm       -      7    -          -          -          armv7"

#      Don't use CGO   Use modules
export CGO_ENABLED="0" GO111MODULE="on"

function build_package(){
		GOOS="$1" \
		GOARCH="$2" \
		GOAMD64="${3#-}" \
		GOARM="${4#-}" \
		GOMIPS="${5#-}" \
		GOMIPS64="${6#-}" \
		GO386="${7#-}" \
		VERBOSE=1 \
		go build \
			-trimpath \
			-ldflags="${go_ldflags}" \
			-o "build/${8}" || \
	{ echo -e "Failed to build current binary."; exit 1; }

	local method
	for method in {"md5","sha1","sha256","sha512"}
	do
		openssl dgst -"${method}" "build/${8}" | sed 's/([^)]*)//g' >> "build/${8}.dgst"
	done
}

echo "${platforms}" | while read -r os arch amd64 arm mips mips64 sse snap
do
	echo -e "[Building] GOOS: ${os} GOARCH: ${arch} GOAMD64: ${amd64} GOARM: ${arm} GOMIPS: ${mips} GOMIPS64: ${mips64} SSE: ${sse} SNAP: ${snap}"
	case "${arch}" in
	"386"|"arm"|"arm64")
		bin_name="gowebdav-${os}-${snap}"
		;;
	"amd64")
		bin_name="gowebdav-${os}-${arch}-${amd64}"
		;;
	"mips"|"mipsle")
		bin_name="gowebdav-${os}-${arch}-${mips}"
		;;
	"mips64"|"mips64le")
		bin_name="gowebdav-${os}-${arch}-${mips64}"
		;;
	*)
		bin_name="gowebdav-${os}-${arch}"
		;;
	esac
	[ "${os}" == "windows" ] && bin_name="${os}-${snap}.exe"
	build_package "${os}" "${arch}" "${amd64}" "${arm}" "${mips}" "${mips64}" "${sse}" "${bin_name}"
done
