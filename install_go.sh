#!/bin/sh
set -eu

PROGRAM_NAME=$(basename "$0")
GO_VERSION=${GO_VERSION:-latest}
INSTALL_ROOT=${INSTALL_ROOT:-/usr/local}
GO_PROXY=${GO_PROXY:-}
DOWNLOAD_BASE=${GO_DOWNLOAD_BASE:-https://go.dev}
CHECK_ONLY=0

usage() {
	cat <<EOF
用法: $PROGRAM_NAME [选项]

从 Go 官方网站安装并校验 Go。默认安装最新稳定版本。

选项:
  --version VERSION       指定版本，例如 1.26.6 或 go1.26.6
  --install-root PATH     安装根目录，默认 /usr/local
  --proxy URL             可选，写入 GOPROXY
  --check                 只检查版本、平台和官方校验值，不安装
  -h, --help              显示帮助

示例:
  sudo sh $PROGRAM_NAME
  sudo sh $PROGRAM_NAME --version 1.26.6
  sudo sh $PROGRAM_NAME --proxy https://goproxy.cn,direct
EOF
}

die() {
	printf '错误: %s\n' "$*" >&2
	exit 1
}

need_command() {
	command -v "$1" >/dev/null 2>&1 || die "缺少命令: $1"
}

run_root() {
	if [ "$(id -u)" -eq 0 ]; then
		"$@"
	else
		need_command sudo
		sudo "$@"
	fi
}

download() {
	download_url=$1
	download_target=$2
	if command -v curl >/dev/null 2>&1; then
		curl -fL --retry 3 --connect-timeout 15 -o "$download_target" "$download_url"
	elif command -v wget >/dev/null 2>&1; then
		wget --tries=3 --timeout=15 -O "$download_target" "$download_url"
	else
		die "需要 curl 或 wget 才能下载 Go"
	fi
}

sha256_file() {
	checksum_target=$1
	if command -v sha256sum >/dev/null 2>&1; then
		sha256sum "$checksum_target" | awk '{print $1}'
	elif command -v shasum >/dev/null 2>&1; then
		shasum -a 256 "$checksum_target" | awk '{print $1}'
	else
		die "需要 sha256sum 或 shasum 才能校验安装包"
	fi
}

while [ "$#" -gt 0 ]; do
	case "$1" in
		--version)
			[ "$#" -ge 2 ] || die "--version 缺少参数"
			GO_VERSION=$2
			shift 2
			;;
		--install-root)
			[ "$#" -ge 2 ] || die "--install-root 缺少参数"
			INSTALL_ROOT=$2
			shift 2
			;;
		--proxy)
			[ "$#" -ge 2 ] || die "--proxy 缺少参数"
			GO_PROXY=$2
			shift 2
			;;
		--check)
			CHECK_ONLY=1
			shift
			;;
		-h|--help)
			usage
			exit 0
			;;
		*)
			die "未知选项: $1（使用 --help 查看帮助）"
			;;
	esac
done

need_command uname
need_command tar
need_command awk
need_command sed
need_command mktemp

case "$(uname -s)" in
	Linux) GOOS=linux ;;
	Darwin) GOOS=darwin ;;
	*) die "暂不支持此操作系统: $(uname -s)" ;;
esac

case "$(uname -m)" in
	x86_64|amd64) GOARCH=amd64 ;;
	aarch64|arm64) GOARCH=arm64 ;;
	armv6l|armv7l) GOARCH=armv6l ;;
	i386|i686) GOARCH=386 ;;
	ppc64le) GOARCH=ppc64le ;;
	riscv64) GOARCH=riscv64 ;;
	s390x) GOARCH=s390x ;;
	loongarch64|loong64) GOARCH=loong64 ;;
	*) die "暂不支持此 CPU 架构: $(uname -m)" ;;
esac

temp_dir=$(mktemp -d "${TMPDIR:-/tmp}/go-install.XXXXXX")
cleanup() {
	rm -rf "$temp_dir"
}
trap cleanup EXIT HUP INT TERM

metadata_file=$temp_dir/releases.json
if [ "$GO_VERSION" = "latest" ]; then
	version_file=$temp_dir/version.txt
	download "$DOWNLOAD_BASE/VERSION?m=text" "$version_file"
	GO_RELEASE=$(sed -n '1p' "$version_file")
	metadata_url=$DOWNLOAD_BASE/dl/?mode=json
else
	case "$GO_VERSION" in
		go*) GO_RELEASE=$GO_VERSION ;;
		*) GO_RELEASE=go$GO_VERSION ;;
	esac
	metadata_url=$DOWNLOAD_BASE/dl/?mode=json\&include=all
fi

case "$GO_RELEASE" in
	go[0-9]*.[0-9]*) ;;
	*) die "无效的 Go 版本: $GO_RELEASE" ;;
esac

archive_name=$GO_RELEASE.$GOOS-$GOARCH.tar.gz
archive_file=$temp_dir/$archive_name
download "$metadata_url" "$metadata_file"

expected_sha256=
found_archive=0
while IFS= read -r metadata_line; do
	case "$metadata_line" in
		*\"filename\":\ \"$archive_name\"*) found_archive=1 ;;
	esac
	if [ "$found_archive" -eq 1 ]; then
		case "$metadata_line" in
			*\"sha256\":*)
				expected_sha256=$(printf '%s\n' "$metadata_line" | sed -n 's/.*"sha256": "\([0-9a-f]*\)".*/\1/p')
				break
				;;
		esac
	fi
done < "$metadata_file"

[ -n "$expected_sha256" ] || die "Go 官方下载列表中没有 $archive_name"

if [ "$CHECK_ONLY" -eq 1 ]; then
	printf '版本: %s\n' "$GO_RELEASE"
	printf '平台: %s/%s\n' "$GOOS" "$GOARCH"
	printf '文件: %s\n' "$archive_name"
	printf 'SHA256: %s\n' "$expected_sha256"
	exit 0
fi

printf '正在下载 %s/%s (%s)...\n' "$GOOS" "$GOARCH" "$GO_RELEASE"
download "$DOWNLOAD_BASE/dl/$archive_name" "$archive_file"

actual_sha256=$(sha256_file "$archive_file")
[ "$actual_sha256" = "$expected_sha256" ] || die "SHA256 校验失败，安装已取消"
printf 'SHA256 校验通过。\n'

tar -xzf "$archive_file" -C "$temp_dir"
[ -x "$temp_dir/go/bin/go" ] || die "安装包内容不完整"

install_dir=$INSTALL_ROOT/go
backup_dir=
if [ -e "$install_dir" ]; then
	backup_dir=$INSTALL_ROOT/go.backup.$(date +%Y%m%d%H%M%S)
	printf '正在备份旧版本到 %s...\n' "$backup_dir"
	run_root mv "$install_dir" "$backup_dir"
fi

if ! run_root mkdir -p "$INSTALL_ROOT" || ! run_root mv "$temp_dir/go" "$install_dir"; then
	if [ -n "$backup_dir" ] && [ -e "$backup_dir" ] && [ ! -e "$install_dir" ]; then
		run_root mv "$backup_dir" "$install_dir"
	fi
	die "安装失败，旧版本已尝试恢复"
fi

profile_file=$temp_dir/go-path.sh
printf 'export PATH="%s/go/bin:$PATH"\n' "$INSTALL_ROOT" > "$profile_file"

if [ "$GOOS" = "darwin" ]; then
	paths_file=$temp_dir/go-path
	printf '%s/go/bin\n' "$INSTALL_ROOT" > "$paths_file"
	run_root install -m 0644 "$paths_file" /etc/paths.d/go
else
	run_root install -m 0644 "$profile_file" /etc/profile.d/go.sh
fi

PATH=$INSTALL_ROOT/go/bin:$PATH
export PATH

if [ -n "$GO_PROXY" ]; then
	go env -w GOPROXY="$GO_PROXY"
fi

printf '\n安装完成。\n'
go version
printf 'GOROOT=%s\n' "$(go env GOROOT)"
printf 'GOPATH=%s\n' "$(go env GOPATH)"
[ -z "$backup_dir" ] || printf '旧版本备份: %s\n' "$backup_dir"
printf '重新登录终端后 PATH 会自动生效。当前终端可执行:\n'
printf '  export PATH="%s/go/bin:\$PATH"\n' "$INSTALL_ROOT"
