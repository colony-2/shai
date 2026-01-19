#!/usr/bin/env bash
set -euo pipefail

VERBOSE=0
SCRIPT_OUTPUT=1

BOOT_SRC_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
SHAI_CONF_DIR=${SHAI_CONF_DIR:-$BOOT_SRC_DIR/conf}
SHAI_RUN_DIR=${SHAI_RUN_DIR:-/run/shai}
SHAI_LOG_DIR=${SHAI_LOG_DIR:-/var/log/shai}
TINYPROXY_CONF_SRC="$SHAI_CONF_DIR/tinyproxy.conf"
DNSMASQ_CONF_SRC="$SHAI_CONF_DIR/dnsmasq.conf"
ALLOWLIST_FILE="$SHAI_RUN_DIR/allowed_domains.conf"
DNS_ALLOW_FILE="$SHAI_RUN_DIR/dnsmasq-allow.conf"
TINYPROXY_CONF="$SHAI_RUN_DIR/tinyproxy.conf"
DNSMASQ_CONF="$SHAI_RUN_DIR/dnsmasq.conf"
TINYPROXY_RUN_DIR="$SHAI_RUN_DIR/tinyproxy"
DNSMASQ_RUN_DIR="$SHAI_RUN_DIR/dnsmasq"
TINYPROXY_PID_FILE="$TINYPROXY_RUN_DIR/tinyproxy.pid"
DNSMASQ_PID_FILE="$DNSMASQ_RUN_DIR/dnsmasq.pid"
PROXY_ENV_FILE="$SHAI_RUN_DIR/proxy-env.sh"
PROFILE_SNIPPET="/etc/profile.d/zz-shai-proxy.sh"
SUPERVISOR_LOG="$SHAI_LOG_DIR/supervisord.log"
SUPERVISOR_PID="$SHAI_RUN_DIR/supervisord.pid"
TINYPROXY_LOG_DIR="$SHAI_LOG_DIR/tinyproxy"
DNSMASQ_LOG_DIR="$SHAI_LOG_DIR/dnsmasq"
TINYPROXY_LOG_FILE="$TINYPROXY_LOG_DIR/tinyproxy.log"
DNSMASQ_LOG_FILE="$DNSMASQ_LOG_DIR/dnsmasq.log"
TINYPROXY_STDOUT_LOG="$TINYPROXY_LOG_DIR/tinyproxy.out.log"
TINYPROXY_STDERR_LOG="$TINYPROXY_LOG_DIR/tinyproxy.err.log"
DNSMASQ_STDOUT_LOG="$DNSMASQ_LOG_DIR/dnsmasq.out.log"
DNSMASQ_STDERR_LOG="$DNSMASQ_LOG_DIR/dnsmasq.err.log"

if [ "${SHAI_SCRIPT_OUTPUT:-1}" = "0" ]; then
  SCRIPT_OUTPUT=0
fi

timestamp() {
  date -Iseconds
}

log() {
  printf '[bootstrap] %s %s\n' "$(timestamp)" "$*" >&2
}

log_verbose() {
  if [ "$VERBOSE" -eq 1 ] && [ "$SCRIPT_OUTPUT" -eq 1 ]; then
    log "$@"
  fi
}

debug() {
  if [ "${SHAI_VERBOSE:-}" = "1" ] && [ "$SCRIPT_OUTPUT" -eq 1 ]; then
    log "$@"
  fi
}

die() {
  log "error: $*"
  exit 90
}

port_in_use() {
  local proto=$1
  local port=$2
  local ss_flag
  case "$proto" in
    tcp) ss_flag="-ltn" ;;
    udp) ss_flag="-lun" ;;
    *) return 1 ;;
  esac

  if command -v ss >/dev/null 2>&1; then
    if ss $ss_flag 2>/dev/null | tail -n +2 | grep -E ":${port}(\\s|$)" >/dev/null 2>&1; then
      return 0
    fi
  elif command -v netstat >/dev/null 2>&1; then
    local netstat_flag="-ltn"
    [ "$proto" = "udp" ] && netstat_flag="-lun"
    if netstat $netstat_flag 2>/dev/null | tail -n +3 | grep -E ":${port}(\\s|$)" >/dev/null 2>&1; then
      return 0
    fi
  elif command -v timeout >/dev/null 2>&1; then
    if timeout 0.25 bash -c "cat < /dev/tcp/127.0.0.1/${port}" >/dev/null 2>&1; then
      return 0
    fi
  fi
  return 1
}

pick_available_port() {
  local start=$1
  local proto=$2
  local port=$start
  while true; do
    case "$proto" in
      tcp)
        if port_in_use tcp "$port"; then
          port=$((port + 1))
          continue
        fi
        ;;
      udp)
        if port_in_use udp "$port"; then
          port=$((port + 1))
          continue
        fi
        ;;
      dns)
        if port_in_use tcp "$port" || port_in_use udp "$port"; then
          port=$((port + 1))
          continue
        fi
        ;;
      *)
        ;;
    esac
    printf '%s\n' "$port"
    return 0
  done
}

render_config() {
  local src=$1
  local dest=$2
  local dir
  dir=$(dirname "$dest")
  mkdir -p "$dir"
  sed \
    -e "s#__PROXY_PORT__#$PROXY_PORT#g" \
    -e "s#__DNS_PORT__#$DNS_PORT#g" \
    -e "s#__RUN_DIR__#$SHAI_RUN_DIR#g" \
    -e "s#__ALLOW_FILE__#$ALLOWLIST_FILE#g" \
    -e "s#__DNS_ALLOW_FILE__#$DNS_ALLOW_FILE#g" \
    -e "s#__CONF_DIR__#$SHAI_CONF_DIR#g" \
    -e "s#__TINY_LOG_FILE__#$TINYPROXY_LOG_FILE#g" \
    -e "s#__DNS_LOG_FILE__#$DNSMASQ_LOG_FILE#g" \
    -e "s#__TINY_PID_FILE__#$TINYPROXY_PID_FILE#g" \
    -e "s#__DNS_PID_FILE__#$DNSMASQ_PID_FILE#g" \
    "$src" >"$dest"
}

find_install_dir() {
  local requested=${SHAI_BOOTSTRAP_INSTALL_DIR:-}
  local -a candidates=()
  local -a path_dirs=()

  if [ -n "$requested" ]; then
    candidates+=("$requested")
  fi

  IFS=':' read -r -a path_dirs <<<"${PATH:-}"
  if [ "${#path_dirs[@]}" -gt 0 ]; then
    candidates+=("${path_dirs[@]}")
  fi

  candidates+=("/usr/local/bin" "/usr/bin" "/bin")

  declare -A seen=()
  for dir in "${candidates[@]}"; do
    [ -z "$dir" ] && continue
    if [ -n "${seen["$dir"]:-}" ]; then
      continue
    fi
    seen["$dir"]=1
    if [ -d "$dir" ] && [ -w "$dir" ]; then
      printf '%s\n' "$dir"
      return 0
    fi
  done
  return 1
}

install_alias_script() {
  local dest_dir
  if ! dest_dir=$(find_install_dir); then
    log_verbose "no writable PATH entry found; skipping shai-remote install"
    return
  fi

  local alias_src="$BOOT_SRC_DIR/shai-remote"
  if [ -f "$alias_src" ]; then
    local alias_dest="$dest_dir/shai-remote"
    if cp "$alias_src" "$alias_dest"; then
      chmod 0755 "$alias_dest" || true
      log_verbose "installed shai-remote to $alias_dest"
    else
      log_verbose "failed to install shai-remote to $alias_dest"
    fi
  else
    log_verbose "shai-remote source missing at $alias_src; nothing to install"
  fi
}

on_exit() {
  if [ "$VERBOSE" -eq 1 ]; then
    status=$?
    log_verbose "bootstrap exiting with status $status"
  fi
}

trap 'on_exit' EXIT
trap 'log_verbose "bootstrap received SIGTERM"; exit 143' TERM
trap 'log_verbose "bootstrap received SIGINT"; exit 130' INT

VERSION=""
TARGET_USER=""
WORKSPACE=""
IMAGE_NAME=""
PROXY_PORT=${PROXY_PORT:-18888}
DNS_PORT=${DNS_PORT:-1053}
REQUESTED_DEV_UID=${DEV_UID:-4747}
REQUESTED_DEV_GID=${DEV_GID:-$REQUESTED_DEV_UID}
RM_SELF="false"

declare -a EXEC_ENVS=()
declare -a EXEC_CMD=()
declare -a HTTP_ALLOW=()
declare -a PORT_ALLOW=()
declare -a RESOURCE_NAMES=()
declare -a ROOT_CMDS=()

require_arg() {
  if [ $# -lt 2 ]; then
    die "flag $1 requires a value"
  fi
}

while [ $# -gt 0 ]; do
  case "$1" in
    --version)
      require_arg "$@"
      VERSION="$2"
      shift 2
      ;;
    --user)
      require_arg "$@"
      TARGET_USER="$2"
      shift 2
      ;;
    --workspace)
      require_arg "$@"
      WORKSPACE="$2"
      shift 2
      ;;
    --exec-env)
      require_arg "$@"
      EXEC_ENVS+=("$2")
      shift 2
      ;;
    --exec-cmd)
      require_arg "$@"
      EXEC_CMD+=("$2")
      shift 2
      ;;
    --http-allow)
      require_arg "$@"
      HTTP_ALLOW+=("$2")
      shift 2
      ;;
    --port-allow)
      require_arg "$@"
      PORT_ALLOW+=("$2")
      shift 2
      ;;
    --rm)
      require_arg "$@"
      RM_SELF="$2"
      shift 2
      ;;
    --image-name)
      require_arg "$@"
      IMAGE_NAME="$2"
      shift 2
      ;;
    --resource-name)
      require_arg "$@"
      RESOURCE_NAMES+=("$2")
      shift 2
      ;;
    --root-cmd)
      require_arg "$@"
      ROOT_CMDS+=("$2")
      shift 2
      ;;
    --verbose)
      VERBOSE=1
      shift
      ;;
    *)
      die "unknown flag $1"
      ;;
  esac
done

[ -z "$VERSION" ] && die "--version is required"
[ -z "$TARGET_USER" ] && die "--user is required"
[ -z "$WORKSPACE" ] && die "--workspace is required"

if [ "$VERSION" != "1" ]; then
  die "unsupported config version $VERSION"
fi

install_alias_script

if [ ! -d "$SHAI_CONF_DIR" ]; then
  die "config directory $SHAI_CONF_DIR missing; bootstrap mount incomplete"
fi

if ! mkdir -p "$SHAI_RUN_DIR"; then
  die "failed to create runtime dir $SHAI_RUN_DIR"
fi
if ! mkdir -p "$SHAI_LOG_DIR"; then
  die "failed to create log dir $SHAI_LOG_DIR"
fi
if ! mkdir -p "$TINYPROXY_LOG_DIR"; then
  die "failed to create tinyproxy log dir $TINYPROXY_LOG_DIR"
fi
if ! mkdir -p "$DNSMASQ_LOG_DIR"; then
  die "failed to create dnsmasq log dir $DNSMASQ_LOG_DIR"
fi
if ! mkdir -p "$TINYPROXY_RUN_DIR"; then
  die "failed to create tinyproxy run dir $TINYPROXY_RUN_DIR"
fi
if ! mkdir -p "$DNSMASQ_RUN_DIR"; then
  die "failed to create dnsmasq run dir $DNSMASQ_RUN_DIR"
fi
if ! mkdir -p "$(dirname "$PROXY_ENV_FILE")"; then
  die "failed to create proxy env dir $(dirname "$PROXY_ENV_FILE")"
fi
touch "$ALLOWLIST_FILE" "$DNS_ALLOW_FILE"

if [ ! -f "$TINYPROXY_CONF_SRC" ]; then
  die "tinyproxy config missing at $TINYPROXY_CONF_SRC"
fi
if [ ! -f "$DNSMASQ_CONF_SRC" ]; then
  die "dnsmasq config missing at $DNSMASQ_CONF_SRC"
fi

PROXY_PORT=$(pick_available_port "$PROXY_PORT" tcp)
DNS_PORT=$(pick_available_port "$DNS_PORT" dns)

render_config "$TINYPROXY_CONF_SRC" "$TINYPROXY_CONF"
render_config "$DNSMASQ_CONF_SRC" "$DNSMASQ_CONF"

if [ "$RM_SELF" = "true" ]; then
  rm -f "$0" 2>/dev/null || true
fi

if [ "$VERBOSE" -eq 1 ]; then
  export SHAI_VERBOSE=1
fi

generate_dnsmasq_allowlist() {
  local allow_file=$1
  local out_file=${2:-$DNS_ALLOW_FILE}
  local upstream4=${UPSTREAM4:-1.1.1.1}
  local upstream4_alt=${UPSTREAM4_ALT:-9.9.9.9}
  local upstream6=${UPSTREAM6:-2606:4700:4700::1111}
  local upstream6_alt=${UPSTREAM6_ALT:-2620:fe::9}

  local tmp
  tmp=$(mktemp)

  {
    echo "# Generated from $allow_file on $(date -u +%FT%TZ)"
    echo "# Forward only listed domains; all others have no upstream and will SERVFAIL"
    while IFS= read -r raw || [ -n "$raw" ]; do
      line="${raw%%#*}"
      line="${line%%$'\r'}"
      line=$(printf '%s' "$line" | awk '{gsub(/^\s+|\s+$/, ""); print}')
      [ -z "$line" ] && continue
      d="$line"
      d="${d#http://}"
      d="${d#https://}"
      d="${d#.}"
      if [[ $d == \*.* ]]; then
        d="${d#\*.}"
      fi
      d=$(printf '%s' "$d" | tr 'A-Z' 'a-z')
      [ -z "$d" ] && continue
      printf "server=/%s/%s\n" "$d" "$upstream4"
      printf "server=/%s/%s\n" "$d" "$upstream4_alt"
      printf "server=/%s/%s\n" "$d" "$upstream6"
      printf "server=/%s/%s\n" "$d" "$upstream6_alt"
    done <"$allow_file"
  } >"$tmp"

  install -m 0644 -D "$tmp" "$out_file"
  rm -f "$tmp"
  log_verbose "wrote dnsmasq allowlist to $out_file"
}

start_supervisord() {
  /usr/bin/supervisord -c /dev/stdin <<SUPERVISOR_CONF
[supervisord]
logfile=$SUPERVISOR_LOG
pidfile=$SUPERVISOR_PID
nodaemon=false
childlogdir=$SHAI_LOG_DIR

[program:tinyproxy]
command=/usr/bin/tinyproxy -d -c $TINYPROXY_CONF
user=tinyproxy
autorestart=true
stdout_logfile=$TINYPROXY_STDOUT_LOG
stderr_logfile=$TINYPROXY_STDERR_LOG

[program:dnsmasq]
command=/usr/sbin/dnsmasq -k --conf-file=$DNSMASQ_CONF
user=root
autorestart=true
stdout_logfile=$DNSMASQ_STDOUT_LOG
stderr_logfile=$DNSMASQ_STDERR_LOG

[include]
files = /etc/supervisor/conf.d/*.conf
SUPERVISOR_CONF
}

compute_docker_host_name() {
  local docker_host_name=${DOCKER_HOST_NAME:-}

  if [ -z "$docker_host_name" ] && [ -n "${SHAI_ALIAS_ENDPOINT:-}" ]; then
    local endpoint=${SHAI_ALIAS_ENDPOINT#*://}
    endpoint=${endpoint%%/*}
    local host_part=${endpoint%%:*}
    host_part=${host_part#[}
    host_part=${host_part%]}
    docker_host_name=${host_part:-host.docker.internal}
  fi

  docker_host_name=${docker_host_name:-host.docker.internal}
  printf '%s\n' "$docker_host_name"
}

dev_egress_setup() {
  local dev_uid=$1
  local proxy_port=$2
  local dns_port=$3
  shift 3
  local port_allow_list=("$@")

  local docker_host_name
  docker_host_name=$(compute_docker_host_name)
  local allow_docker_host_port=${ALLOW_DOCKER_HOST_PORT:-}

  if [ -n "$allow_docker_host_port" ]; then
    local alias_entry="${docker_host_name}:${allow_docker_host_port}"
    local found=0
    for existing in "${port_allow_list[@]}"; do
      if [ "$existing" = "$alias_entry" ]; then
        found=1
        break
      fi
    done
    if [ "$found" -eq 0 ]; then
      port_allow_list+=("$alias_entry")
    fi
  fi

  ensure_rule() {
    local table=$1
    shift
    if ! iptables -t "$table" -C "$@" 2>/dev/null; then
      iptables -t "$table" -A "$@"
    fi
  }

  ensure_rule6() {
    local table=$1
    shift
    if ! ip6tables -t "$table" -C "$@" 2>/dev/null; then
      ip6tables -t "$table" -A "$@"
    fi
  }

  resolve_host_ip() {
    local name=$1
    local ip=""
    local ping_output=""

    if command -v ping >/dev/null 2>&1; then
      if ping_output=$(ping -c1 -W1 "$name" 2>/dev/null); then
        ip=$(printf '%s' "$ping_output" | sed -n '1s/[^(]*(\([^)]*\)).*/\1/p')
      fi
    fi

    if [ -z "$ip" ] && command -v getent >/dev/null 2>&1; then
      ip=$(getent hosts "$name" | awk 'NR==1 {print $1; exit}')
    fi

    echo "$ip"
  }

  if command -v iptables >/dev/null 2>&1; then
    local docker_host_ip
    docker_host_ip=$(resolve_host_ip "$docker_host_name")
    if [ -n "$docker_host_ip" ]; then
      ensure_rule filter OUTPUT -m owner --uid-owner "$dev_uid" -p tcp -d "$docker_host_ip" -j ACCEPT
      ensure_rule filter OUTPUT -m owner --uid-owner "$dev_uid" -p udp -d "$docker_host_ip" -j ACCEPT
    fi

    ensure_rule filter OUTPUT -m owner --uid-owner "$dev_uid" -o lo -j ACCEPT
    ensure_rule filter OUTPUT -m owner --uid-owner "$dev_uid" -p tcp -d 127.0.0.1 --dport "$proxy_port" -j ACCEPT
    # Force all DNS requests through the local dnsmasq instance.
    ensure_rule nat OUTPUT -m owner --uid-owner "$dev_uid" -p udp --dport 53 -j REDIRECT --to-ports "$dns_port"
    ensure_rule nat OUTPUT -m owner --uid-owner "$dev_uid" -p tcp --dport 53 -j REDIRECT --to-ports "$dns_port"
    ensure_rule nat OUTPUT -m owner --uid-owner "$dev_uid" -p udp --dport "$dns_port" -j REDIRECT --to-ports "$dns_port"
    ensure_rule nat OUTPUT -m owner --uid-owner "$dev_uid" -p tcp --dport "$dns_port" -j REDIRECT --to-ports "$dns_port"
    ensure_rule filter OUTPUT -m owner --uid-owner "$dev_uid" -p udp -d 127.0.0.1 --dport "$dns_port" -j ACCEPT
    ensure_rule filter OUTPUT -m owner --uid-owner "$dev_uid" -p tcp -d 127.0.0.1 --dport "$dns_port" -j ACCEPT

    for entry in "${port_allow_list[@]}"; do
      local host=${entry%%:*}
      local port=${entry##*:}
      if [ -z "$host" ] || [ -z "$port" ]; then
        continue
      fi
      local host_ip
      host_ip=$(resolve_host_ip "$host")
      if [ -n "$host_ip" ]; then
        log_verbose "allowing tcp ${host}:${port} (${host_ip})"
        ensure_rule filter OUTPUT -m owner --uid-owner "$dev_uid" -p tcp -d "$host_ip" --dport "$port" -j ACCEPT
      else
        log_verbose "warning: unable to resolve $host, allowing port $port without destination restriction"
        ensure_rule filter OUTPUT -m owner --uid-owner "$dev_uid" -p tcp --dport "$port" -j ACCEPT
      fi
    done

    ensure_rule filter OUTPUT -m owner --uid-owner "$dev_uid" -j REJECT
    if [ "$VERBOSE" -eq 1 ]; then
      iptables -S OUTPUT || true
    fi
  fi

  if command -v ip6tables >/dev/null 2>&1; then
    if ip6tables -t nat -L OUTPUT >/dev/null 2>&1; then
      ensure_rule6 nat OUTPUT -m owner --uid-owner "$dev_uid" -p udp --dport 53 -j REDIRECT --to-ports "$dns_port"
      ensure_rule6 nat OUTPUT -m owner --uid-owner "$dev_uid" -p tcp --dport 53 -j REDIRECT --to-ports "$dns_port"
      ensure_rule6 nat OUTPUT -m owner --uid-owner "$dev_uid" -p udp --dport "$dns_port" -j REDIRECT --to-ports "$dns_port"
      ensure_rule6 nat OUTPUT -m owner --uid-owner "$dev_uid" -p tcp --dport "$dns_port" -j REDIRECT --to-ports "$dns_port"
    else
      log_verbose "ip6tables nat table unavailable; skipping IPv6 DNS redirect"
    fi
    ensure_rule6 filter OUTPUT -m owner --uid-owner "$dev_uid" -p tcp -d ::1 --dport "$proxy_port" -j ACCEPT
    ensure_rule6 filter OUTPUT -m owner --uid-owner "$dev_uid" -p udp -d ::1 --dport "$dns_port" -j ACCEPT
    ensure_rule6 filter OUTPUT -m owner --uid-owner "$dev_uid" -p tcp -d ::1 --dport "$dns_port" -j ACCEPT
    ensure_rule6 filter OUTPUT -m owner --uid-owner "$dev_uid" -j REJECT
    if [ "$VERBOSE" -eq 1 ]; then
      ip6tables -S OUTPUT || true
    fi
  fi

  local log_dir="/var/log/shai"
  local log_file="$log_dir/iptables.out"
  mkdir -p "$log_dir" 2>/dev/null || true
  {
    echo "# iptables rules (generated at $(date))"
    echo "# IPv4 filter table OUTPUT chain:"
    if command -v iptables >/dev/null 2>&1; then
      iptables -t filter -S OUTPUT 2>/dev/null || echo "# Failed to dump IPv4 filter rules"
    fi
    echo ""
    echo "# IPv4 nat table OUTPUT chain:"
    if command -v iptables >/dev/null 2>&1; then
      iptables -t nat -S OUTPUT 2>/dev/null || echo "# Failed to dump IPv4 nat rules"
    fi
    echo ""
    echo "# IPv6 filter table OUTPUT chain:"
    if command -v ip6tables >/dev/null 2>&1; then
      ip6tables -t filter -S OUTPUT 2>/dev/null || echo "# Failed to dump IPv6 filter rules"
    fi
    echo ""
    echo "# IPv6 nat table OUTPUT chain:"
    if command -v ip6tables >/dev/null 2>&1; then
      ip6tables -t nat -S OUTPUT 2>/dev/null || echo "# Failed to dump IPv6 nat rules"
    fi
  } >"$log_file" 2>/dev/null || true
  chmod 644 "$log_file" 2>/dev/null || true
  log_verbose "iptables rules logged to $log_file"
}
require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    die "required command $1 not found"
  fi
}

user_by_uid() {
  local uid=$1
  if [ -z "$uid" ]; then
    return
  fi
  getent passwd | awk -F: -v uid="$uid" '$3==uid {print $1; exit}'
}

group_by_gid() {
  local gid=$1
  if [ -z "$gid" ]; then
    return
  fi
  getent group | awk -F: -v gid="$gid" '$3==gid {print $1; exit}'
}

ensure_group_with_gid() {
  local gid=$1
  local preferred=$2
  if [ -z "$gid" ]; then
    return
  fi

  local existing
  existing=$(group_by_gid "$gid")
  if [ -n "$existing" ]; then
    printf '%s\n' "$existing"
    return
  fi

  require_cmd groupadd

  local name=$preferred
  if [ -z "$name" ]; then
    name="shai"
  fi
  if getent group "$name" >/dev/null 2>&1; then
    name="shai-${gid}"
  fi

  # Suppress GID range warnings since we intentionally use host GID
  err_output=$(groupadd -g "$gid" "$name" 2>&1)
  status=$?
  # Show errors except UID/GID range warnings (grep returns 1 if all lines filtered, so use || true)
  echo "$err_output" | grep -vE "warning:.*outside of the (UID|GID)_MIN.*_MAX.*range" >&2 || true
  if [ $status -eq 0 ]; then
    printf '%s\n' "$name"
    return
  fi

  die "failed to create group with gid $gid"
}

reconcile_target_user() {
  local requested_uid=$REQUESTED_DEV_UID
  local requested_gid=$REQUESTED_DEV_GID
  local uid_user=""
  local gid_group=""

  if [ -n "$requested_uid" ]; then
    uid_user=$(user_by_uid "$requested_uid")
  fi
  if [ -n "$requested_gid" ]; then
    gid_group=$(group_by_gid "$requested_gid")
  fi

  if [ "$IS_ROOT" -ne 1 ]; then
    if ! id "$TARGET_USER" >/dev/null 2>&1; then
      die "user $TARGET_USER does not exist and bootstrap cannot create it without root"
    fi
    DEV_UID=$(id -u "$TARGET_USER")
    DEV_GID=$(id -g "$TARGET_USER")
    if [ -n "$requested_uid" ] && [ "$DEV_UID" != "$requested_uid" ]; then
      die "user $TARGET_USER has uid $DEV_UID but host uid $requested_uid; rerun with root"
    fi
    if [ -n "$requested_gid" ] && [ "$DEV_GID" != "$requested_gid" ]; then
      die "user $TARGET_USER has gid $DEV_GID but host gid $requested_gid; rerun with root"
    fi
    return
  fi

  if [ -n "$requested_uid" ] && [ -z "$uid_user" ]; then
    log_verbose "no existing account with uid $requested_uid; will align $TARGET_USER to host uid"
  fi

  if ! id "$TARGET_USER" >/dev/null 2>&1; then
    if [ -n "$uid_user" ]; then
      log_verbose "reusing existing user $uid_user with uid $requested_uid"
      TARGET_USER="$uid_user"
    else
      require_cmd useradd
      local home_dir="/home/$TARGET_USER"
      local args=()
      if [ -d "$home_dir" ]; then
        args+=(-M)
      else
        args+=(-m)
      fi
      if [ -n "$requested_uid" ]; then
        args+=(-u "$requested_uid")
      fi
      local primary_group=""
      if [ -n "$requested_gid" ]; then
        primary_group=$(ensure_group_with_gid "$requested_gid" "$TARGET_USER")
      fi
      if [ -n "$primary_group" ]; then
        args+=(-g "$primary_group")
      fi
      args+=(-d "$home_dir" -s /bin/bash)
      # Suppress UID range warnings since we intentionally use host UID
      err_output=$(useradd "${args[@]}" "$TARGET_USER" 2>&1)
      status=$?
      # Show errors except UID/GID range warnings (grep returns 1 if all lines filtered, so use || true)
      echo "$err_output" | grep -vE "warning:.*outside of the (UID|GID)_MIN.*_MAX.*range" >&2 || true
      if [ $status -ne 0 ]; then
        die "failed to create user $TARGET_USER"
      fi
    fi
  else
    local current_uid
    current_uid=$(id -u "$TARGET_USER")
    if [ -n "$requested_uid" ] && [ "$current_uid" != "$requested_uid" ]; then
      if [ -n "$uid_user" ] && [ "$uid_user" != "$TARGET_USER" ]; then
        log_verbose "target user $TARGET_USER uses uid $current_uid; switching to existing uid-matched user $uid_user"
        TARGET_USER="$uid_user"
      else
        require_cmd usermod
        if ! usermod -u "$requested_uid" "$TARGET_USER"; then
          die "failed to update uid for $TARGET_USER to $requested_uid"
        fi
      fi
    fi
  fi

  DEV_UID=$(id -u "$TARGET_USER")
  DEV_GID=$(id -g "$TARGET_USER")

  if [ -n "$requested_gid" ] && [ "$DEV_GID" != "$requested_gid" ]; then
    local target_group=$gid_group
    if [ -z "$target_group" ]; then
      target_group=$(ensure_group_with_gid "$requested_gid" "$TARGET_USER")
    fi
    if [ -z "$target_group" ]; then
      die "unable to resolve group for gid $requested_gid"
    fi
    require_cmd usermod
    if ! usermod -g "$target_group" "$TARGET_USER"; then
      die "failed to update primary group for $TARGET_USER to gid $requested_gid"
    fi
    DEV_GID=$(id -g "$TARGET_USER")
  fi

  # Ensure home directory ownership is correct after any user/group changes
  # Note: Not recursive to avoid failures on read-only mounted subdirectories
  local user_home
  user_home=$(getent passwd "$TARGET_USER" | cut -d: -f6)
  if [ -n "$user_home" ] && [ -d "$user_home" ]; then
    # Use "user:" syntax to set user and their primary group (not "user:user")
    if ! chown "$TARGET_USER:" "$user_home"; then
      die "failed to set ownership of $user_home to $TARGET_USER (uid=$DEV_UID, gid=$DEV_GID)"
    fi
  fi

  export TARGET_USER
  export DEV_UID DEV_GID
}

main() {
  IS_ROOT=1
  if [ "${EUID:-$(id -u)}" -ne 0 ]; then
    IS_ROOT=0
    debug "running without root privileges; skipping privileged setup"
  fi

  reconcile_target_user

  log_verbose "bootstrap start (uid=${EUID:-$(id -u)}, dev_uid=$DEV_UID, dev_gid=$DEV_GID, proxy_port=$PROXY_PORT)"

  local docker_host_name
  docker_host_name=$(compute_docker_host_name)

  local allow_hosts=()
  if [ ${#HTTP_ALLOW[@]} -gt 0 ]; then
    allow_hosts=("${HTTP_ALLOW[@]}")
  fi
  for entry in "${PORT_ALLOW[@]}"; do
    local host=${entry%%:*}
    [ -n "$host" ] && allow_hosts+=("$host")
  done
  if [ -n "${ALLOW_DOCKER_HOST_PORT:-}" ]; then
    allow_hosts+=("$docker_host_name")
  fi
  if [ ${#allow_hosts[@]} -gt 0 ]; then
    # de-duplicate
    local uniq_hosts=()
    declare -A seen_hosts=()
    for h in "${allow_hosts[@]}"; do
      [ -z "$h" ] && continue
      if [ -z "${seen_hosts[$h]:-}" ]; then
        uniq_hosts+=("$h")
        seen_hosts[$h]=1
      fi
    done
    allow_hosts=("${uniq_hosts[@]}")
  fi

  mkdir -p "$(dirname "$ALLOWLIST_FILE")"
  if [ ${#allow_hosts[@]} -gt 0 ]; then
    printf '%s\n' "${allow_hosts[@]}" >"$ALLOWLIST_FILE"
    debug "updated tinyproxy allowlist with ${#allow_hosts[@]} entries"
  else
    : >"$ALLOWLIST_FILE"
    debug "tinyproxy allowlist empty; http proxy will deny all outbound traffic"
  fi
  generate_dnsmasq_allowlist "$ALLOWLIST_FILE" "$DNS_ALLOW_FILE"

  if [ "$IS_ROOT" -eq 1 ]; then
    debug "ensuring log directories"
    mkdir -p "$SHAI_LOG_DIR" "$TINYPROXY_LOG_DIR" "$DNSMASQ_LOG_DIR"
    mkdir -p /etc/supervisor/conf.d
    chown tinyproxy:tinyproxy "$TINYPROXY_LOG_DIR" "$TINYPROXY_RUN_DIR" 2>/dev/null || true

    SUP_PIDFILE=$SUPERVISOR_PID
    if ! [ -f "$SUP_PIDFILE" ] || ! kill -0 "$(cat "$SUP_PIDFILE" 2>/dev/null)" 2>/dev/null; then
      log_verbose "starting supervisord"
      if start_supervisord; then
        debug "supervisord started"
      else
        status=$?
        die "supervisord launch exited with status $status"
      fi
    else
      debug "supervisord already running (pid $(cat "$SUP_PIDFILE" 2>/dev/null))"
    fi
  fi

  if [ ${#PORT_ALLOW[@]} -gt 0 ]; then
    dev_egress_setup "$DEV_UID" "$PROXY_PORT" "$DNS_PORT" "${PORT_ALLOW[@]}"
  else
    dev_egress_setup "$DEV_UID" "$PROXY_PORT" "$DNS_PORT"
  fi

  if [ ! -d "$WORKSPACE" ]; then
    debug "workspace $WORKSPACE does not exist; attempting to create"
    mkdir -p "$WORKSPACE" 2>/dev/null || true
  fi

  user_entry=$(getent passwd "$TARGET_USER" || true)
  if [ -z "$user_entry" ]; then
    die "user $TARGET_USER not found in passwd database"
  fi
  user_home=$(printf '%s\n' "$user_entry" | cut -d: -f6)
  user_shell=$(printf '%s\n' "$user_entry" | cut -d: -f7)
  if [ -z "$user_shell" ]; then
    user_shell="/bin/bash"
  fi

  export HOME="$user_home"
  export USER="$TARGET_USER"
  export WORKSPACE="$WORKSPACE"
  export SHAI_WORKSPACE="$WORKSPACE"

  proxy_url="http://127.0.0.1:${PROXY_PORT}"
  no_proxy="localhost,127.0.0.1,::1"

  cat >"$PROXY_ENV_FILE" <<EOF
export HTTP_PROXY="$proxy_url"
export HTTPS_PROXY="$proxy_url"
export http_proxy="$proxy_url"
export https_proxy="$proxy_url"
export NO_PROXY="$no_proxy"
export no_proxy="$no_proxy"
EOF
  chmod 0644 "$PROXY_ENV_FILE"

  export HTTP_PROXY="$proxy_url"
  export HTTPS_PROXY="$proxy_url"
  export http_proxy="$proxy_url"
  export https_proxy="$proxy_url"
  if [ -n "$docker_host_name" ]; then
    if ! printf '%s' "$no_proxy" | tr ',' '\n' | grep -qxF "$docker_host_name"; then
      no_proxy="$no_proxy,$docker_host_name"
    fi
    if host_ip=$(getent hosts "$docker_host_name" 2>/dev/null | awk '{print $1; exit}'); then
      if ! printf '%s' "$no_proxy" | tr ',' '\n' | grep -qxF "$host_ip"; then
        no_proxy="$no_proxy,$host_ip"
      fi
    fi
  fi
  export NO_PROXY="$no_proxy"
  export no_proxy="$no_proxy"
  export BASH_ENV="$PROXY_ENV_FILE"
  export ENV="$PROXY_ENV_FILE"

  if [ "$IS_ROOT" -eq 1 ]; then
    mkdir -p "$(dirname "$PROFILE_SNIPPET")"
    cat >"$PROFILE_SNIPPET" <<EOF
if [ -f "$PROXY_ENV_FILE" ]; then
  . "$PROXY_ENV_FILE"
fi
EOF
    chmod 0644 "$PROFILE_SNIPPET"

    if [ -f /etc/zsh/zshenv ]; then
      if ! grep -q "$PROXY_ENV_FILE" /etc/zsh/zshenv 2>/dev/null; then
        cat >>/etc/zsh/zshenv <<EOF

# Source proxy configuration from shai bootstrap
if [ -f "$PROXY_ENV_FILE" ]; then
  . "$PROXY_ENV_FILE"
fi
EOF
      fi
    fi
  fi

  for pair in "${EXEC_ENVS[@]}"; do
    key=${pair%%=*}
    value=${pair#*=}
    if [ -z "$key" ]; then
      continue
    fi
    export "$key"="$value"
  done

  cd "$WORKSPACE" 2>/dev/null || die "failed to enter workspace $WORKSPACE"

  if [ "$IS_ROOT" -eq 1 ] && [ ${#ROOT_CMDS[@]} -gt 0 ]; then
    log_verbose "executing ${#ROOT_CMDS[@]} root command(s)"
    for cmd in "${ROOT_CMDS[@]}"; do
      if [ -n "$cmd" ]; then
        log_verbose "executing root command: $cmd"
        if ! eval "$cmd"; then
          die "root command failed (exit $?): $cmd"
        fi
      fi
    done
  fi

  argv=("${EXEC_CMD[@]}")
  if [ ${#argv[@]} -eq 0 ]; then
    argv=("$user_shell" "-l")
  fi

  resource_summary="none"
  if [ ${#RESOURCE_NAMES[@]} -gt 0 ]; then
    resource_summary=$(printf '%s, ' "${RESOURCE_NAMES[@]}")
    resource_summary=${resource_summary%, }
  fi
  image_desc=${IMAGE_NAME:-unknown}
  summary_message="Shai sandbox started using [$image_desc] as user [$TARGET_USER]. Resource sets: [$resource_summary]"

  if [ "$SCRIPT_OUTPUT" -eq 1 ]; then
    printf '%s\n' "$summary_message"
  fi

  if [ "$IS_ROOT" -eq 1 ]; then
    if ! command -v runuser >/dev/null 2>&1; then
      die "runuser not found (required from util-linux package)"
    fi
    exec runuser -u "$TARGET_USER" -- "${argv[@]}"
  else
    if [ "${EUID:-$(id -u)}" -ne "$DEV_UID" ]; then
      die "bootstrap not running as $TARGET_USER and cannot switch users"
    fi
    exec "${argv[@]}"
  fi
}

main
  
