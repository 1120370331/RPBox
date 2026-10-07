#!/usr/bin/env bash
set -euo pipefail
cd "$HOME/RPBox"
test "$(git rev-parse HEAD)" = eb9462c06af755e2548b1365222fa56f9ba07248
incoming="$HOME/rpbox-ta6-public-release"
test -f "$incoming/rpbox-server"
test -f "$incoming/manifest.json"
cd "$incoming"
sha256sum -c binary.sha256
test ! -e applied.json
before_pid="$(sudo supervisorctl pid rpbox)"
test "$before_pid" -gt 0
test "$(readlink /proc/$before_pid/exe)" = "$HOME/RPBox/server/rpbox-server"
stable_before="$(sha256sum "$HOME/RPBox/server/releases/latest.json" | cut -d' ' -f1)"
previous_sha="$(sha256sum "$HOME/RPBox/server/rpbox-server" | cut -d' ' -f1)"
cp -p "$HOME/RPBox/server/rpbox-server" previous-rpbox-server
cp -p "$HOME/RPBox/server/releases/latest.json" previous-stable-metadata.json
cp rpbox-server "$HOME/RPBox/server/rpbox-server.ta6-new"
chmod 755 "$HOME/RPBox/server/rpbox-server.ta6-new"
mv "$HOME/RPBox/server/rpbox-server.ta6-new" "$HOME/RPBox/server/rpbox-server"
rollback() {
  cp previous-rpbox-server "$HOME/RPBox/server/rpbox-server.ta6-rollback"
  chmod 755 "$HOME/RPBox/server/rpbox-server.ta6-rollback"
  mv "$HOME/RPBox/server/rpbox-server.ta6-rollback" "$HOME/RPBox/server/rpbox-server"
  sudo supervisorctl restart rpbox
}
if ! sudo supervisorctl restart rpbox; then rollback; exit 1; fi
port="$(awk '/^server:/{inside=1; next} inside && /^[^[:space:]]/{exit} inside && /^[[:space:]]+port:/{print $2; exit}' "$HOME/RPBox/server/config.yaml" | tr -d '\"')"
port="${port:-8080}"
healthy=0
for attempt in $(seq 1 30); do
  status="$(curl -sS -o /dev/null -w '%{http_code}' -H 'X-Forwarded-Proto: https' "http://127.0.0.1:$port/api/v1/updater/windows/x86_64/0.0.0" || true)"
  if [ "$status" = 200 ]; then healthy=1; break; fi
  sleep 2
done
if [ "$healthy" != 1 ]; then rollback; exit 1; fi
after_pid="$(sudo supervisorctl pid rpbox)"
actual_sha="$(sha256sum /proc/$after_pid/exe | cut -d' ' -f1)"
expected_sha="$(cut -d' ' -f1 binary.sha256)"
if [ "$actual_sha" != "$expected_sha" ]; then rollback; exit 1; fi
test "$(sha256sum "$HOME/RPBox/server/releases/latest.json" | cut -d' ' -f1)" = "$stable_before"
export before_pid after_pid previous_sha actual_sha stable_before
python3 - <<'PY'
import datetime,json,os,pathlib
manifest=json.loads(pathlib.Path('manifest.json').read_text())
manifest.update({'deployedUTC':datetime.datetime.now(datetime.timezone.utc).isoformat(),
 'beforePID':int(os.environ['before_pid']),'afterPID':int(os.environ['after_pid']),
 'previousBinarySHA256':os.environ['previous_sha'],'runningBinarySHA256':os.environ['actual_sha'],
 'stableMetadataSHA256':os.environ['stable_before'],'localHealthStatus':200,
 'databaseReset':False,'sourceCheckoutChanged':False,
 'rollback':'Copy ~/rpbox-ta6-public-release/previous-rpbox-server to ~/RPBox/server/rpbox-server using a temporary file then supervisorctl restart rpbox'})
pathlib.Path('applied.json').write_text(json.dumps(manifest,indent=2))
print('TA6_DEPLOYMENT_EVIDENCE='+json.dumps(manifest))
PY
