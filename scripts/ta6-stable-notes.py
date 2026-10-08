import base64, hashlib, json, os, shlex, subprocess
from pathlib import Path

assert os.environ['VERSION'] == '1.0.5'
assert os.environ['CHANNEL'] == 'stable'
assert os.environ['METADATA_FILE'] == 'latest.json'
old = json.loads(Path('current-latest.json').read_text())
assert old['latest_version'] == '1.0.5' and old['channel'] == 'stable'
assert old['pub_date'] == '2026-10-08T06:08:38Z'
new = dict(old)
new['notes'] = Path('client/release-notes/1.0.5.txt').read_text().strip()
assert '及个人资料中展示' not in new['notes']
assert '并在公会详情中展示' in new['notes']
payload = json.dumps(new, ensure_ascii=False).encode()
encoded = base64.b64encode(payload).decode()
remote = r"""
import base64, hashlib, json, os, pathlib, sys
base = pathlib.Path(sys.argv[1]).resolve()
payload = base64.b64decode(sys.argv[2])
run = sys.argv[3]
assert run.isdigit()
paths = [base/'latest.json', base/'1.0.5/latest.json']
original = [p.read_bytes() for p in paths]
desired = json.loads(payload)
assert desired['latest_version'] == '1.0.5' and desired['channel'] == 'stable'
assert desired['pub_date'] == '2026-10-08T06:08:38Z'
assert '及个人资料中展示' not in desired['notes']
assert '并在公会详情中展示' in desired['notes']
installer = base/'1.0.5/RPBox_1.0.5_x64-setup.exe'
signature = pathlib.Path(str(installer)+'.sig')
def sha(path): return hashlib.sha256(path.read_bytes()).hexdigest()
protected = {str(installer): '5b74bcdb23fa52672538ff5587c0dd788bfc31d7a0e582727181940d49181acc',
             str(signature): 'e3d038b4dad31339602c5ce82b2ce27bb21e2a98a2bf5f32cb99c5120f05d6b9'}
beta = base/'latest-beta.json'
if beta.exists(): protected[str(beta)] = sha(beta)
assert all(sha(pathlib.Path(p)) == h for p,h in protected.items())
for data in original:
    prior = json.loads(data)
    assert {k:v for k,v in prior.items() if k != 'notes'} == {k:v for k,v in desired.items() if k != 'notes'}
    assert prior['notes'] == desired['notes'] or hashlib.sha256(data).hexdigest() == '32e7c84a9fcddc65d4a205cf9757818b0c088261e500cbffdad120fa852f7543'
backup = base/'1.0.5'/('.ta6-notes-backup-'+run)
backup.mkdir(exist_ok=True)
for i,data in enumerate(original):
    target=backup/str(i)
    if target.exists(): assert target.read_bytes()==data
    else: target.write_bytes(data)
changed=[]
try:
    for path,data in zip(paths,original):
        assert path.read_bytes()==data
        temp=path.with_name(path.name+'.ta6-notes-'+run)
        with temp.open('xb') as handle:
            handle.write(payload); handle.flush(); os.fsync(handle.fileno())
        os.chmod(temp, path.stat().st_mode & 0o777)
        os.replace(temp,path); changed.append((path,data))
    assert all(p.read_bytes()==payload for p in paths)
    assert all(sha(pathlib.Path(p))==h for p,h in protected.items())
except BaseException:
    for path,data in reversed(changed):
        if path.read_bytes()==payload:
            temp=path.with_name(path.name+'.ta6-rollback-'+run)
            temp.write_bytes(data); os.replace(temp,path)
    raise
print(json.dumps({'version':'1.0.5','channel':'stable','metadataSHA256':hashlib.sha256(payload).hexdigest(),
                  'installerAndSignatureUnchanged':True,'betaUnchanged':True,'pubDateUnchanged':True}))
"""
host=os.environ['SERVER_USER']+'@'+os.environ['SERVER_HOST']
args=[os.environ['REMOTE_BASE'],encoded,os.environ['GITHUB_RUN_ID']]
remote_command='python3 -c '+shlex.quote(remote)+' '+ ' '.join(shlex.quote(x) for x in args)
result=subprocess.run(['ssh','-p','2233','-o','ConnectTimeout=30',host,remote_command],capture_output=True,check=True)
Path('metadata-repair-result.json').write_bytes(result.stdout)
print(result.stdout.decode())
