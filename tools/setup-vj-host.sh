#!/usr/bin/env bash
# setup-vj-host.sh — Ubuntu 24.04 を kagelife の VJ 機にする。何度流しても同じ結果になる。
#
# やること:
#   1. 軽いデスクトップ（Xorg + Openbox + LightDM）と、確認用の道具を入れる
#   2. VJ_USER で自動ログインし、画面の消灯とスリープを止める
#   3. kagelife のリリース（linux-amd64）を取得し、SHA256SUMS で確かめて入れる
#   4. kagelife-shaders を取得する
#   5. ログインしたら kagelife を全画面で起動する
#
# 使い方: sudo VJ_USER=<ユーザー> bash setup-vj-host.sh
#   KAGELIFE_VERSION（既定 v0.1.0）、SHADERS_REF（既定 main）、PRESET（既定 basic-vj）、
#   KAGELIFE_ARGS（kagelife に足す引数、既定は空）で変えられる。
#   例: sudo VJ_USER=cyokozai KAGELIFE_ARGS='-render-scale 0.5' bash setup-vj-host.sh
#   引数を変えたら sudo systemctl restart lightdm でログインからやり直す
#   （動いている start.sh は前の引数のまま kagelife を起動し直すため）。
#
# 実機の記録（2026-11-11、HP ProDesk 600 G3 SFF・Intel HD Graphics 630・1920x1080）:
#   basic-vj は 04_warp 以外の 8 本が 60fps。04_warp は -render-scale 1.0 で 24fps、0.75 で 39fps、0.5 で 60fps。
set -euo pipefail

VJ_USER="${VJ_USER:?VJ_USER を指定してください}"
KAGELIFE_VERSION="${KAGELIFE_VERSION:-v0.1.0}"
SHADERS_REF="${SHADERS_REF:-main}"
PRESET="${PRESET:-basic-vj}"
KAGELIFE_ARGS="${KAGELIFE_ARGS:-}"

[ "$(id -u)" -eq 0 ] || { echo "root で実行してください（sudo）" >&2; exit 1; }
id "$VJ_USER" >/dev/null 2>&1 || { echo "ユーザー $VJ_USER がいません" >&2; exit 1; }
HOME_DIR="$(getent passwd "$VJ_USER" | cut -d: -f6)"
as_user() { sudo -u "$VJ_USER" -H "$@"; }

echo "== 1. パッケージ"
export DEBIAN_FRONTEND=noninteractive
apt-get update -q
apt-get install -y -q --no-install-recommends \
  xserver-xorg xinit x11-xserver-utils \
  openbox lightdm lightdm-gtk-greeter \
  libgl1 libgl1-mesa-dri mesa-utils libasound2t64 alsa-utils \
  scrot unclutter curl ca-certificates git

echo "== 2. 自動ログインと画面・スリープ"
install -d /etc/lightdm/lightdm.conf.d
cat > /etc/lightdm/lightdm.conf.d/50-vj-autologin.conf <<EOF
[Seat:*]
autologin-user=$VJ_USER
autologin-user-timeout=0
user-session=openbox
EOF
groupadd -f autologin
usermod -aG autologin,audio,video "$VJ_USER"
systemctl mask sleep.target suspend.target hibernate.target hybrid-sleep.target >/dev/null
systemctl set-default graphical.target >/dev/null
systemctl enable lightdm >/dev/null 2>&1 || true

echo "== 3. kagelife $KAGELIFE_VERSION"
APP_DIR="$HOME_DIR/Applications/kagelife"
ASSET="kagelife-$KAGELIFE_VERSION-linux-amd64"
as_user mkdir -p "$APP_DIR"
if [ ! -x "$APP_DIR/$ASSET/kagelife" ]; then
  BASE="https://github.com/cyokozai/kagelife/releases/download/$KAGELIFE_VERSION"
  as_user curl -fsSL -o "$APP_DIR/$ASSET.tar.gz" "$BASE/$ASSET.tar.gz"
  as_user curl -fsSL -o "$APP_DIR/SHA256SUMS" "$BASE/SHA256SUMS"
  (cd "$APP_DIR" && grep " $ASSET.tar.gz\$" SHA256SUMS | sha256sum -c -)
  as_user tar -xzf "$APP_DIR/$ASSET.tar.gz" -C "$APP_DIR"
fi
as_user ln -sfn "$APP_DIR/$ASSET" "$APP_DIR/current"
as_user "$APP_DIR/current/kagelife" -version

echo "== 4. kagelife-shaders ($SHADERS_REF)"
SH_DIR="$HOME_DIR/kagelife-shaders"
if [ ! -d "$SH_DIR/.git" ]; then
  as_user git clone -q https://github.com/cyokozai/kagelife-shaders.git "$SH_DIR"
fi
as_user git -C "$SH_DIR" fetch -q origin
as_user git -C "$SH_DIR" checkout -q --detach "origin/$SHADERS_REF"
as_user git -C "$SH_DIR" log --oneline -1

echo "== 5. 起動時に kagelife を全画面で"
VJ_DIR="$HOME_DIR/vj"
as_user mkdir -p "$VJ_DIR" "$HOME_DIR/.config/openbox"
cat > "$VJ_DIR/start.sh" <<EOF
#!/usr/bin/env bash
# setup-vj-host.sh が書いた。kagelife を全画面で起動し、落ちたら 3 秒後に起動し直す。
cd "$VJ_DIR"
while true; do
  "$APP_DIR/current/kagelife" -fullscreen -shaders "$SH_DIR/presets/$PRESET" $KAGELIFE_ARGS >>"$VJ_DIR/kagelife.log" 2>&1
  echo "\$(date '+%F %T') kagelife exited (\$?), restarting" >>"$VJ_DIR/kagelife.log"
  sleep 3
done
EOF
cat > "$HOME_DIR/.config/openbox/autostart" <<EOF
# setup-vj-host.sh が書いた
xset s off; xset s noblank; xset -dpms
unclutter -idle 1 &
"$VJ_DIR/start.sh" &
EOF
chown "$VJ_USER:$VJ_USER" "$VJ_DIR/start.sh" "$HOME_DIR/.config/openbox/autostart"
chmod 755 "$VJ_DIR/start.sh"

echo "== 完了。sudo systemctl restart lightdm（または再起動）で全画面の kagelife が上がる"
