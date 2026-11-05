"""Local screenshot fixtures; never contacts or writes the production database.

Run the client with VITE_API_BASE=http://127.0.0.1:9017/api/v1 on Vite port 3188,
then run this script and open http://127.0.0.1:9017/market (or /settings,
/community, /sync?mode=cloud, /archives/story/1). The Vue application and theme
come directly from client/src; only transport data and native shell calls are mocked.
"""
import base64
import json
from datetime import datetime, timezone, timedelta
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path
from urllib.parse import urlparse, parse_qs

ROOT = Path(__file__).resolve().parents[1]
ASSETS = ROOT / 'style-demos/assets'
NOW = datetime.now(timezone.utc).isoformat()
GAME_PATH = 'C:/Games/World of Warcraft/_retail_'
USER = dict(id=991, username='Orphy', email='preview@example.invalid', role='user',
            is_sponsor=False, sponsor_level=0, avatar='http://127.0.0.1:9017/api/v1/images/avatar/1', created_at=NOW,
            activity_experience=0, forum_level=1, forum_level_name='初识艾泽拉斯')
TAGS = [dict(id=i+1, name=n, color=c, category='item', is_preset=True)
        for i, (n, c) in enumerate([('冒险', 'C9A36D'), ('日常', '8DAB82'), ('故事', 'B9A0C7')])]
ITEMS = [dict(id=i+1, author_id=991, author_username=author, name=name, type=kind,
              description=desc, preview_image_url='fixture', preview_image_updated_at=NOW,
              downloads=86+i*19, rating=4.8, rating_count=12+i, like_count=24+i*3,
              favorite_count=18+i, status='published', is_public=True,
              review_status='approved', created_at=NOW, updated_at=NOW)
         for i, (name, kind, desc, author) in enumerate([
             ('群山之外的旅人手札', 'document', '沿着旧地图，记录那些没有名字的道路与偶然相逢的人。', 'Orphy'),
             ('炉火旁的约定', 'campaign', '一次酒馆偶遇，三封未寄出的信，一场关于归途的冒险。', '木槿'),
             ('远行者的魔法行囊', 'item', '为长途旅行准备的随身收纳，装下地图、故事与一点点惊喜。', '风栖'),
             ('荒野寻踪', 'campaign', '适合小队参与的原野探索剧本，循着足迹寻找失落的营地。', '长夜旅人'),
             ('酒馆纪事：团务手册', 'document', '整理角色关系、旅途线索与团务记录的随身小册子。', '木槿'),
             ('黄昏信使', 'item', '一封写给远方的信，承载一个尚未结束的故事。', 'Orphy'),
         ])]
POSTS = [dict(id=i+1, author_id=991+i, author_name=author, title=title,
              content=content, content_type='html', category=category,
              is_pinned=False, is_featured=i==0, is_public=True, status='published',
              review_status='approved', cover_image_url='fixture', view_count=128+i*24,
              like_count=18+i*6, comment_count=6+i*2, favorite_count=12,
              created_at=NOW, updated_at=NOW)
         for i, (title, content, category, author) in enumerate([
             ('群山之外，还有新的故事', '<p>我们带着旧地图出发，在原野上发现了一条通往群山的小径。旅途从来不缺故事，只缺愿意驻足倾听的人。</p>', 'novel', 'Orphy'),
             ('炉火不熄，旅人常来', '<p>今晚的酒馆为远行的人留了一张空椅。欢迎分享旅途中的见闻，或者一起喝杯热茶。</p>', 'guild', '木槿'),
             ('分享我的新角色：风栖·林歌', '<p>一位在林间长大的游侠。整理了背景、性格和初见印象，期待遇见志同道合的旅伴。</p>', 'profile', '风栖'),
             ('把昨晚的冒险留在这里', '<p>从落日出发，到星光下的营火。为同行的朋友留下一份完整的剧情记录。</p>', 'report', '长夜旅人'),
         ])]
PROFILES = {f'profile-{i+1}': {'profileName': n, 'player': {'characteristics': {'FN': n, 'RA': race, 'CL': job}}}
            for i, (n, race, job) in enumerate([('Orphy · 行路者', '人类', '探险家'), ('风栖 · 林歌', '暗夜精灵', '游侠'), ('木槿 · 炉边旅人', '人类', '吟游诗人')])}
TOOLS = {f'item-{i+1}': {'MD': {'NA': name}, 'TY': 'item'} for i, name in enumerate(['旅人的旧地图', '群山之外的手札', '远行者的魔法行囊', '黄昏信使'])}
BACKUP = dict(id=1, user_id=991, account_id='远行者的旅途', game_flavor='_retail_',
              profiles_data=json.dumps(PROFILES, ensure_ascii=False), profiles_count=3,
              tools_count=len(TOOLS), tools_data=json.dumps(TOOLS, ensure_ascii=False), runtime_size_kb=248,
              extra_data=json.dumps({'TRP3_Characters': {'Orphy': 'profile-1'}, 'TRP3_Companions': {'小旅伴': '猫'}, 'TRP3_Notes': {'营地': '群山脚下'}}),
              checksum='p1p2p3toolsruntimeconfigextra', version=6, created_at=NOW, updated_at=NOW)
LOCAL_ACCOUNT = dict(account_id=BACKUP['account_id'], profiles=[
    dict(id=key, name=value['profileName'], checksum=f'p{i+1}', raw_lua='',
         account_id=BACKUP['account_id'], saved_variables_path=GAME_PATH+'/WTF/Account/Adventurer/SavedVariables')
    for i, (key,value) in enumerate(PROFILES.items())],
    tools_db=dict(item_count=len(TOOLS), checksum='tools', raw_data=BACKUP['tools_data']),
    runtime_data=dict(size_kb=248, checksum='runtime', raw_data='{}'),
    config=dict(checksum='config', raw_data='{}'),
    extra_data=dict(checksum='extra', raw_data=BACKUP['extra_data']))
EVENT = dict(POSTS[1], id=90, title='炉火不熄 · 旅人故事之夜', category='event',
             event_type='server', event_color='#B68A50',
             event_start_time=(datetime.now(timezone.utc)-timedelta(hours=1)).isoformat(),
             event_end_time=(datetime.now(timezone.utc)+timedelta(hours=3)).isoformat(),
             cover_image_url='http://127.0.0.1:9017/api/v1/images/post-cover/2',
             region='艾尔文森林', address='狮王之傲旅店')
STORY = dict(id=1, user_id=991, title='群山之外，还有新的故事',
             description='原野上的一次相遇，和一段刚刚开始的旅途。', region='艾尔文森林', address='林间营地',
             participants='Orphy,风栖,木槿', tags='冒险,日常', status='published',
             is_public=True, share_code='local-preview', entry_count=8, view_count=0,
             start_time=NOW, end_time=NOW, created_at=NOW, updated_at=NOW)
CHARACTERS = [dict(
    id=i+1, user_id=991, ref_id=f'preview-character-{i+1}', game_id=name,
    is_npc=False, created_at=NOW, updated_at=NOW, trp3_version=1,
    race=race, **{'class': job}, first_name=name, last_name='', full_title=job,
    title='', icon='', color=color, eye_color='', age='', height='',
    residence='', birthplace='', misc_info='{}', psycho='{}', about_text='{}',
    custom_avatar=f'http://127.0.0.1:9017/api/v1/images/avatar/{i+1}',
    custom_name='', custom_color='', raw_trp3_data='{}')
    for i, (name, race, job, color) in enumerate([
        ('Orphy', '人类', '探险家', 'D6B57F'),
        ('风栖', '暗夜精灵', '游侠', 'B6C6E6'),
        ('木槿', '人类', '吟游诗人', 'D9AA88'),
    ])]
CHARACTER_IDS = {character['first_name']: character['id'] for character in CHARACTERS}
for item in ITEMS:
    item['author_avatar'] = f"http://127.0.0.1:9017/api/v1/images/avatar/{CHARACTER_IDS.get(item['author_username'], 3)}"
for post in POSTS:
    post['author_avatar'] = f"http://127.0.0.1:9017/api/v1/images/avatar/{CHARACTER_IDS.get(post['author_name'], 3)}"
ENTRIES = [dict(id=i+1, story_id=1, source_id=f'preview-{i}', type=kind,
                speaker=name, content=content, channel='EMOTE' if kind=='narration' else 'SAY',
                timestamp=NOW, sort_order=i, character_id=CHARACTER_IDS[name])
           for i, (name, kind, content) in enumerate([
               ('Orphy', 'dialogue', '你看见那条路了吗？它一定通向什么地方。'),
               ('风栖', 'narration', '她顺着登山杖指向的方向望去，远处的群山正在夕阳中泛起金色。'),
               ('风栖', 'dialogue', '也许是另一座城镇，也许是一段没有人讲过的故事。'),
               ('Orphy', 'narration', '他扶了扶帽檐，掌心的魔盒泛起微光，映亮了摊开的旧地图。'),
               ('木槿', 'dialogue', '那么，就把今晚写作第一章吧。'),
               ('Orphy', 'dialogue', '走吧。世界比我们想象的还要辽阔。'),
               ('风栖', 'narration', '三人的脚步汇入小径。晚风掠过原野，带来远方篝火的气息。'),
               ('木槿', 'dialogue', '等我们回来，再把这一路的故事讲给大家听。'),
           ])]

def bootstrap():
    token = 'preview.' + base64.b64encode(json.dumps({'exp': 4102444800}).encode()).decode() + '.local-only'
    setup = {'token': token, 'user': json.dumps(USER, ensure_ascii=False), 'theme': 'dark-coffee',
             'locale': 'zh-CN', 'last_viewed_version': '1.0.6-6', 'addon_auto_update_enabled': 'false',
             'wow_path': GAME_PATH, 'selected_flavor': '_retail_'}
    return '''<!doctype html><html lang="zh-CN"><head><meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>RPBox · 深色咖啡预览</title></head><body><div id="app"></div><script>
    const settings=SETTINGS;for(const [key,value] of Object.entries(settings))localStorage.setItem(key,value);
    let callbackId=0;
    window.__TAURI_INTERNALS__={
      metadata:{currentWindow:{label:'main'},currentWebview:{label:'main'}},
      transformCallback(callback){const id=++callbackId;window['_'+id]=callback;return id},
      unregisterCallback(id){delete window['_'+id]},
      invoke:async command=>{
        if(command==='scan_profiles')return {accounts:[LOCAL_ACCOUNT]};
        if(command==='plugin:app|version')return '1.0.6-6';
        if(command==='plugin:autostart|is_enabled')return false;
        if(command==='plugin:deep-link|get_current')return [];
        if(command==='plugin:event|listen')return 1;
        if(command.includes('list_')||command.includes('scan_'))return [];
        return null;
      }
    };
    const RealSocket=window.WebSocket;
    window.WebSocket=class extends EventTarget {
      static OPEN=1;static CLOSED=3;static CONNECTING=0;readyState=1;
      constructor(url,protocols){super();if(!String(url).includes('/ws/notifications'))return new RealSocket(url,protocols);setTimeout(()=>this.onopen?.({}),0)}
      send(){}close(){this.readyState=3}
    };
    </script><script type="module" src="http://127.0.0.1:3188/@vite/client"></script><script type="module" src="http://127.0.0.1:3188/src/main.ts"></script></body></html>'''.replace('SETTINGS', json.dumps(setup, ensure_ascii=False)).replace('LOCAL_ACCOUNT', json.dumps(LOCAL_ACCOUNT, ensure_ascii=False))

class Handler(BaseHTTPRequestHandler):
    def respond(self, data, content_type='application/json; charset=utf-8', status=200):
        payload = data if isinstance(data, bytes) else (data.encode() if isinstance(data, str) else json.dumps(data, ensure_ascii=False).encode())
        self.send_response(status)
        self.send_header('Content-Type', content_type)
        self.send_header('Access-Control-Allow-Origin', '*')
        self.send_header('Content-Length', str(len(payload)))
        self.end_headers()
        self.wfile.write(payload)

    def do_OPTIONS(self):
        self.send_response(204)
        self.send_header('Access-Control-Allow-Origin', '*')
        self.send_header('Access-Control-Allow-Headers', 'Authorization, Content-Type')
        self.send_header('Access-Control-Allow-Methods', 'GET, OPTIONS')
        self.end_headers()

    def do_GET(self):
        url = urlparse(self.path)
        path = url.path.removeprefix('/api/v1')
        if not url.path.startswith('/api/v1'):
            if url.path.startswith(('/@fs/', '/node_modules/', '/src/', '/@vite/', '/assets/')):
                self.send_response(302)
                self.send_header('Location', 'http://127.0.0.1:3188' + self.path)
                self.end_headers()
                return
            return self.respond(bootstrap(), 'text/html; charset=utf-8')
        if path.startswith('/images/'):
            group, image_id = path.split('/')[-2:]
            avatar_files = {'1': 'avatar-orphy.png', '2': 'avatar-fengqi.png', '3': 'avatar-mujin.png'}
            item_files = {'1': 'cover-journal.png', '2': 'cover-tavern.png', '3': 'cover-satchel.png', '4': 'cover-wilderness.png', '5': 'cover-journal.png', '6': 'cover-satchel.png'}
            post_files = {'1': 'cover-wilderness.png', '2': 'cover-tavern.png', '3': 'avatar-fengqi.png', '4': 'cover-journal.png'}
            file = (avatar_files if group == 'avatar' else item_files if group == 'item-preview' else post_files).get(image_id, 'cover-tavern.png')
            asset = ASSETS / file
            if not asset.is_file():
                return self.respond({'error': 'Local preview image is unavailable'}, status=404)
            return self.respond(asset.read_bytes(), 'image/png')
        if path == '/user/info': data = USER
        elif path == '/items': data = {'code': 0, 'data': {'items': ITEMS, 'total': len(ITEMS)}}
        elif path == '/posts':
            posts = [] if parse_qs(url.query).get('is_pinned') == ['true'] else POSTS
            data = {'posts': posts, 'total': len(posts)}
        elif path == '/account-backups': data = {'backups': [BACKUP]}
        elif path.startswith('/account-backups/'):
            data = {'versions': [dict(id=i, backup_id=1, version=i, checksum=f'preview-checksum-{i}', name='旅途出发前的备份' if i==6 else '人物卡更新', profiles_count=3, tools_count=12, created_at=NOW, change_log='manual') for i in [6,5,4]]} if path.endswith('/versions') else BACKUP
        elif path == '/stories/1': data = {'story': STORY, 'entries': ENTRIES, 'character_cards': {}}
        elif path == '/stories': data = {'stories': [STORY]}
        elif path.endswith('/bookmarks'): data = {'bookmarks': []}
        elif path.endswith('/music'): data = {'tracks': [], 'segments': [], 'playlists': []}
        elif path.endswith('/tags') or path == '/tags/preset': data = {'tags': TAGS}
        elif path.endswith('/guilds'): data = {'guilds': []}
        elif path == '/characters': data = {'characters': CHARACTERS}
        elif path.startswith('/characters/') and path.rsplit('/',1)[-1] in ['1', '2', '3']:
            data = CHARACTERS[int(path.rsplit('/',1)[-1])-1]
        elif path.startswith('/character-cards'): data = {'cards': [], 'total': 0}
        elif path == '/sponsors': data = {'users': json.loads((ROOT/'server/storage/sponsors/extra_sponsors.json').read_text(encoding='utf-8-sig'))}
        elif path == '/updater/latest': data = {'latest_version': '1.0.6-6', 'version': '1.0.6-6', 'notes': '', 'pub_date': NOW}
        elif path == '/addon/latest': data = {'version': '1.0.0', 'addons': []}
        elif path == '/addon/manifest': data = {'version': '1.0.0', 'addons': []}
        elif path == '/addon/trp3/latest': data = {'addons': []}
        elif path == '/rpdb/works': data = {'works': [], 'total': 0}
        elif path in ['/events', '/posts/events']: data = {'events': [EVENT]}
        elif path == '/notifications/unread-count': data = {'count': 0}
        else: data = {'items': [], 'posts': [], 'events': [], 'total': 0}
        self.respond(data)

    def do_POST(self):
        self.respond({'error': 'Screenshot fixtures are read-only'}, status=405)

    do_PUT = do_DELETE = do_POST

if __name__ == '__main__':
    print('Real client UI + local fixtures: http://127.0.0.1:9017/market', flush=True)
    ThreadingHTTPServer(('127.0.0.1', 9017), Handler).serve_forever()
