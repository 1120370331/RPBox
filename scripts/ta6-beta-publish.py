"""Publish only client beta files using the established release SSH facility."""
import datetime, hashlib, json, os, pathlib, re, shlex, subprocess

version = os.environ['VERSION']
assert version == '1.0.5-14'
base = os.environ['RELEASE_PATH'].rstrip('/')
assert base.startswith('/') and base.endswith('/server/releases') and '..' not in base.split('/')
host = os.environ['SERVER_USER'] + '@' + os.environ['SERVER_HOST']
ssh = ['ssh', '-p', '2233', '-o', 'ConnectTimeout=30', '-o', 'ServerAliveInterval=30', host]
scp = ['scp', '-P', '2233', '-o', 'ConnectTimeout=30', '-o', 'ServerAliveInterval=30']
installers = list(pathlib.Path('artifacts').rglob(f'RPBox_{version}_x64-setup.exe'))
assert len(installers) == 1
exe = installers[0]
sig = pathlib.Path(str(exe) + '.sig')
assert sig.is_file()
def digest(data):
    return hashlib.sha256(data).hexdigest()
notes = pathlib.Path(f'client/release-notes/beta/{version}.txt').read_text(encoding='utf-8')
url = f'https://ksxvodevhonx.sealosbja.site/releases/{version}/{exe.name}'
metadata = {'version': version, 'latest_version': version, 'channel': 'beta', 'notes': notes,
            'pub_date': datetime.datetime.now(datetime.timezone.utc).strftime('%Y-%m-%dT%H:%M:%SZ'),
            'source_commit': os.environ['GITHUB_SHA'], 'url': url,
            'sha256': digest(exe.read_bytes()),
            'platforms': {'windows-x86_64': {'url': url, 'signature': sig.read_text().strip()}}}
meta_file = pathlib.Path('beta-publication-metadata.json')
meta_file.write_text(json.dumps(metadata, ensure_ascii=False, indent=2) + '\n', encoding='utf-8')
hashes = {exe.name: digest(exe.read_bytes()), sig.name: digest(sig.read_bytes()), 'latest.json': digest(meta_file.read_bytes())}
stage = f'.ta6-stage-{version}-{os.environ["GITHUB_RUN_ID"]}-{os.environ["GITHUB_RUN_ATTEMPT"]}'

remote = r'''
import fcntl, hashlib, json, os, pathlib, sys
base, version, stage, operation, expected_json, stable_before, beta_before = sys.argv[1:]
base = pathlib.Path(base)
assert base.is_absolute() and str(base).endswith('/server/releases') and base.resolve() == base
target = base/version
staging = base/stage
assert staging.parent == base and target.parent == base
expected = json.loads(expected_json)
def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest() if path.exists() else None
def state():
    beta = base/'latest-beta.json'
    return {'stableSHA256': sha(base/'latest.json'), 'betaSHA256': sha(beta),
            'betaMetadata': json.loads(beta.read_text()) if beta.exists() else None,
            'versionDirectoryExists': target.exists()}
if operation == 'prepare':
    before = state()
    assert before['stableSHA256'], 'Stable metadata missing; do not publish'
    assert not target.exists(), 'Version occupied; do not overwrite existing release'
    beta = before['betaMetadata']
    if beta:
        current = beta.get('latest_version', beta.get('version', ''))
        def parts(v):
            return tuple(int(x) for x in v.replace('-', '.').split('.'))
        assert parts(current) < parts(version), 'Do not downgrade concurrent beta publication'
    staging.mkdir(mode=0o755, exist_ok=False)
    print(json.dumps(before))
elif operation == 'commit':
    with open(base/'.ta6-beta-publish.lock', 'a') as lock:
        fcntl.flock(lock, fcntl.LOCK_EX)
        before = state()
        assert before['stableSHA256'] == stable_before, 'Stable changed concurrently; stop'
        assert before['betaSHA256'] == (beta_before or None), 'Beta changed concurrently; stop'
        assert not target.exists(), 'Version occupied concurrently; stop'
        for name, expected_hash in expected.items():
            assert pathlib.Path(name).name == name and sha(staging/name) == expected_hash
        meta = json.loads((staging/'latest.json').read_text())
        assert meta['version'] == version and meta['channel'] == 'beta'
        assert meta['sha256'] == expected['RPBox_'+version+'_x64-setup.exe']
        os.rename(staging, target)
        tmp = base/(stage+'.metadata')
        tmp.write_bytes((target/'latest.json').read_bytes())
        os.chmod(tmp, 0o644)
        os.replace(tmp, base/'latest-beta.json')
        after = state()
        assert after['stableSHA256'] == stable_before
        assert after['betaSHA256'] == expected['latest.json']
        print(json.dumps({'published': True, 'version': version, 'directory': str(target),
              'stableUnchanged': True, 'serverDeployed': False, 'fileHashes': expected,
              'betaMetadata': after['betaMetadata'], 'stableSHA256': after['stableSHA256'],
              'betaSHA256': after['betaSHA256']}))
else:
    raise AssertionError('Unknown operation')
'''
def run_remote(operation, stable='', beta=''):
    args = ['python3', '-', base, version, stage, operation, json.dumps(hashes), stable, beta]
    result = subprocess.run(ssh + [shlex.join(args)], input=remote, capture_output=True, text=True, check=True)
    return json.loads(result.stdout)
before = run_remote('prepare')
pathlib.Path('beta-publication-before.json').write_text(json.dumps(before, indent=2), encoding='utf-8')
for path, name in [(exe, exe.name), (sig, sig.name), (meta_file, 'latest.json')]:
    subprocess.run(scp + [str(path), host + ':' + shlex.quote(base + '/' + stage + '/' + name)], check=True)
after = run_remote('commit', before['stableSHA256'], before['betaSHA256'] or '')
pathlib.Path('beta-publication-after.json').write_text(json.dumps(after, ensure_ascii=False, indent=2), encoding='utf-8')
print(json.dumps({'published': True, 'version': version, 'url': url, 'sha256': hashes[exe.name],
                  'stableUnchanged': after['stableUnchanged'], 'serverDeployed': False}))
