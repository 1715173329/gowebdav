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
android    arm64     0      0    0          0          0          aarch64
darwin     amd64     v1     0    0          0          0          0
darwin     amd64     v2     0    0          0          0          0
darwin     amd64     v3     0    0          0          0          0
darwin     amd64     v4     0    0          0          0          0
darwin     arm64     0      0    0          0          0          aarch64
dragonfly  amd64     0      0    0          0          0          0
freebsd    386       0      0    0          0          softfloat  386-softfloat
freebsd    386       0      0    0          0          sse2       386-sse2
freebsd    amd64     0      0    0          0          0          0
freebsd    arm       0      7    0          0          0          armv7
freebsd    arm64     0      0    0          0          0          aarch64
linux      386       0      0    0          0          softfloat  386-softfloat
linux      386       0      0    0          0          sse2       386-sse2
linux      amd64     v1     0    0          0          0          0
linux      amd64     v2     0    0          0          0          0
linux      amd64     v3     0    0          0          0          0
linux      amd64     v4     0    0          0          0          0
linux      arm       0      5    0          0          0          armv5
linux      arm       0      6    0          0          0          armv6
linux      arm       0      7    0          0          0          armv7
linux      arm64     0      0    0          0          0          aarch64
linux      mips      0      0    hardfloat  0          0          0
linux      mips      0      0    softfloat  0          0          0
linux      mips64    0      0    0          hardfloat  0          0
linux      mips64    0      0    0          softfloat  0          0
linux      mips64le  0      0    0          hardfloat  0          0
linux      mips64le  0      0    0          softfloat  0          0
linux      mipsle    0      0    hardfloat  0          0          0
linux      mipsle    0      0    softfloat  0          0          0
linux      ppc64     0      0    0          0          0          0
linux      ppc64le   0      0    0          0          0          0
linux      riscv64   0      0    0          0          0          0
linux      s390x     0      0    0          0          0          0
openbsd    arm       0      7    0          0          0          armv7
openbsd    arm64     0      0    0          0          0          aarch64
openbsd    386       0      0    0          0          softfloat  386-softfloat
openbsd    386       0      0    0          0          sse2       386-sse2
openbsd    amd64     v1     0    0          0          0          0
openbsd    amd64     v2     0    0          0          0          0
openbsd    amd64     v3     0    0          0          0          0
openbsd    amd64     v4     0    0          0          0          0
windows    386       0      0    0          0          softfloat  x86_softfloat
windows    386       0      0    0          0          sse2       x86_sse2
windows    amd64     v1     0    0          0          0          x86_64_v1
windows    amd64     v2     0    0          0          0          x86_64_v2
windows    amd64     v3     0    0          0          0          x86_64_v3
windows    amd64     v4     0    0          0          0          x86_64_v4
windows    arm       0      6    0          0          0          armv6
windows    arm       0      7    0          0          0          armv7"

#      Don't use CGO   Use modules
export CGO_ENABLED="0" GO111MODULE="on"

function build_package(){
	env \
		GOOS="$1" \
		GOARCH="$2" \
		GOAMD64="{3#0}"
		GOARM="${4#0}" \
		GOMIPS="${5#0}" \
		GOMIPS64="${6#0}" \
		GO386="${7#0}" \
		go build \
			-trimpath \
			-ldflags="${go_ldflags}" \
			-o "build/${bin_name}" || \
	{ echo -e "Failed to build current binary."; exit 1; }

	local method
	for method in {"md5","sha1","sha256","sha512"}
	do
		openssl dgst -"${method}" "build/${bin_name}" | sed 's/([^)]*)//g' >> "build/${bin_name}.dgst"
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
	build_package "${os}" "${arch}" "${arm}" "${mips}" "${mips64}" "${sse}"
done
