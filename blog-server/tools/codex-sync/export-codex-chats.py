"""Export an owner-authorized, text-only snapshot through Codex's local app-server.

No model requests, resume operations, local database reads, or uploads are performed.
Upload the generated JSON using the owner's AI workspace after signing in.
"""
import argparse
import datetime as dt
import json
import os
from pathlib import Path
import queue
import re
import shutil
import subprocess
import sys
import threading
import tomllib

MAX_FILE = 8 << 20
ALLOWED_NAMES = {'yia-blog', 'yia-project'}
SECRET = re.compile(r'(?i)sk-[\w-]{12,}|eyJ[\w-]{10,}\.[\w-]{10,}\.[\w-]{10,}|(?:api[_-]?key|password|secret|authorization|(?:access|refresh)[_-]?token)\s*[:=]\s*[\"\']?(?:Bearer\s+)?[\w./+-]{16,}')


def redact(text, secrets=()):
    for secret in sorted(secrets, key=len, reverse=True):
        text = text.replace(secret, '[凭据已隐藏]')
    return SECRET.sub('[凭据已隐藏]', text)


def local_secrets(home):
    """Only compare credential values in memory. Never export auth/config files."""
    found = set()
    def scan(value):
        if isinstance(value, dict):
            for key, item in value.items():
                if re.search(r'(?i)key|token|secret|password', key) and isinstance(item, str) and len(item) >= 12:
                    found.add(item)
                else:
                    scan(item)
        elif isinstance(value, list):
            for item in value:
                scan(item)
    for name in ('auth.json', 'config.toml'):
        p = home / name
        if p.is_file():
            data = p.read_text(encoding='utf-8')
            scan(json.loads(data) if name.endswith('.json') else tomllib.loads(data))
    return found


def text_messages(thread, secrets=()):
    result = []
    seen = set()
    for turn in thread.get('turns', []):
        for item in turn.get('items', []):
            role = None
            if item.get('type') == 'userMessage':
                role = 'user'
                text = '\n'.join(part.get('text', '') for part in item.get('content', []) if part.get('type') == 'text')
                if text.lstrip().startswith('# AGENTS.md instructions'):
                    continue
                for tag in ('environment_context', 'permissions', 'app_context', 'turn_aborted', 'INSTRUCTIONS'):
                    text = re.sub(r'<'+tag+r'\b[^>]*>.*?</'+tag+r'>', '', text, flags=re.S)
                if text.lstrip().startswith('<send_user_message_question_reply>'):
                    text = re.sub(r'</?send_user_message_question_reply>', '', text)
            elif item.get('type') == 'agentMessage':
                role = 'assistant'
                text = item.get('text', '')
            else:
                continue
            text = redact(text.strip(), secrets)
            if not text:
                continue
            if len(text.encode('utf-8')) > 100000:
                text = text.encode('utf-8')[:99900].decode('utf-8', errors='ignore') + '\n[文字过长，副本已截断]'
            identity = str(item.get('id', ''))
            if not identity or identity in seen or len(identity) > 128:
                continue
            seen.add(identity)
            # app-server's stored items do not consistently expose per-message timestamps.
            result.append({'id': identity, 'role': role, 'content': text, 'create_at': utc_time(thread['updatedAt'])})
    return result


def utc_time(seconds):
    return dt.datetime.fromtimestamp(seconds, dt.timezone.utc).isoformat().replace('+00:00', 'Z')


class AppServer:
    def __init__(self, executable, cwd):
        self.process = subprocess.Popen([executable, 'app-server'], cwd=cwd, stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.DEVNULL, text=True, encoding='utf-8')
        self.responses = queue.Queue()
        self.sequence = 0
        def read():
            for line in self.process.stdout:
                try:
                    self.responses.put(json.loads(line))
                except ValueError:
                    pass
            self.responses.put(None)
        threading.Thread(target=read, daemon=True).start()
        self.call('initialize', {'clientInfo': {'name': 'yia_chat_export', 'version': '1.0.0'}, 'capabilities': {'experimentalApi': True}})
        self.notify('initialized', {})

    def notify(self, method, params):
        self.process.stdin.write(json.dumps({'method': method, 'params': params})+'\n')
        self.process.stdin.flush()

    def call(self, method, params):
        self.sequence += 1
        identity = self.sequence
        self.process.stdin.write(json.dumps({'id': identity, 'method': method, 'params': params})+'\n')
        self.process.stdin.flush()
        while True:
            value = self.responses.get(timeout=45)
            if value is None:
                raise RuntimeError('Codex app-server ended before completing a read')
            if value.get('id') != identity:
                continue
            if 'error' in value:
                # Do not echo errors that might contain a local path or credential.
                raise RuntimeError('Codex read method unavailable: '+method)
            return value['result']

    def close(self):
        self.process.stdin.close()
        try:
            self.process.wait(timeout=5)
        except subprocess.TimeoutExpired:
            self.process.terminate()
            self.process.wait(timeout=5)


def roots_for(projects):
    roots = {}
    for root in projects:
        root = Path(root).resolve(strict=True)
        if root.name not in ALLOWED_NAMES:
            raise ValueError('Only yia-blog / yia-project roots are allowed')
        roots[os.path.normcase(str(root))] = root.name
        # Match exact registered worktrees; never select a different project by title alone.
        process = subprocess.run(['git', '-C', str(root), 'worktree', 'list', '--porcelain'], capture_output=True, text=True, encoding='utf-8')
        if process.returncode == 0:
            for line in process.stdout.splitlines():
                if line.startswith('worktree '):
                    path = Path(line[9:]).resolve(strict=True)
                    roots[os.path.normcase(str(path))] = root.name
    return roots


def find_codex():
    direct = shutil.which('codex.exe')
    if direct:
        return direct
    appdata = Path(os.environ.get('APPDATA', ''))
    candidates = list((appdata/'npm/node_modules/@openai/codex/node_modules/@openai').glob('codex-win32-*/vendor/*/bin/codex.exe'))
    if len(candidates) == 1:
        return str(candidates[0])
    direct = shutil.which('codex')
    if direct and not direct.endswith(('.ps1', '.cmd')):
        return direct
    raise RuntimeError('Codex executable unavailable; provide --codex with its executable path')


def export(server, roots, secrets, limit=200, selected=()):
    threads = []
    skipped = 0
    ids = set()
    for archived in (False, True):
        cursor = None
        while len(threads) < limit:
            page = server.call('thread/list', {'cwd': list(roots), 'cursor': cursor, 'limit': 50, 'sortKey': 'updated_at', 'archived': archived, 'modelProviders': [], 'sourceKinds': ['cli', 'vscode', 'appServer'], 'useStateDbOnly': True})
            for summary in page['data']:
                cwd = os.path.normcase(str(Path(summary['cwd']).resolve()))
                identity = summary['id']
                if cwd not in roots or summary.get('parentThreadId') or identity in ids or (selected and identity not in selected):
                    continue
                ids.add(identity)
                # This installed Codex version exposes includeTurns. It never resumes a thread.
                thread = server.call('thread/read', {'threadId': identity, 'includeTurns': True})['thread']
                actual_cwd = os.path.normcase(str(Path(thread['cwd']).resolve()))
                if actual_cwd != cwd:
                    raise RuntimeError('Thread workspace changed during export')
                messages = text_messages(thread, secrets)
                if not messages:
                    continue
                value = {'id': identity, 'title': redact((thread.get('name') or summary.get('name') or summary.get('preview') or 'Codex conversation')[:200], secrets), 'workspace': roots[cwd], 'updated_at': utc_time(thread['updatedAt']), 'messages': messages}
                if len(messages) > 20000 or len(json.dumps(value, ensure_ascii=False).encode('utf-8')) > MAX_FILE-16384:
                    skipped += 1
                    continue
                threads.append(value)
                if len(threads) >= limit:
                    break
            cursor = page.get('nextCursor')
            if not cursor:
                break
    return {'version': 1, 'threads': threads}, skipped


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--project', action='append', required=True, help='Exact local project root (repeat for both projects)')
    parser.add_argument('--output', required=True, help='New private JSON file; existing files are never overwritten')
    parser.add_argument('--thread', action='append', default=[], help='Optional explicit thread ID filter')
    parser.add_argument('--limit', type=int, default=200)
    parser.add_argument('--codex')
    args = parser.parse_args()
    if not 1 <= args.limit <= 200:
        raise ValueError('Limit must be 1..200')
    target = Path(args.output).resolve()
    # Keep snapshots outside tracked source and public assets.
    project_roots = [Path(p).resolve() for p in args.project]
    if not any(target.is_relative_to(p/'.review-commit-prep') for p in project_roots):
        raise ValueError('Output must be inside a project .review-commit-prep directory')
    if target.exists():
        raise ValueError('Output already exists; choose a new snapshot name')
    roots = roots_for(args.project)
    secrets = local_secrets(Path(os.environ.get('CODEX_HOME', str(Path.home()/'.codex'))))
    server = AppServer(args.codex or find_codex(), project_roots[0])
    try:
        bundle, skipped = export(server, roots, secrets, args.limit, args.thread)
    finally:
        server.close()
    if not bundle['threads']:
        raise RuntimeError('No eligible text conversations found; no empty snapshot was written')
    data = json.dumps(bundle, ensure_ascii=False).encode('utf-8')
    if len(data) > 50 << 20:
        raise RuntimeError('Snapshot over 50MB; export fewer threads using --limit or --thread')
    target.parent.mkdir(parents=True, exist_ok=True)
    with target.open('xb') as file:
        os.chmod(target, 0o600)
        file.write(data)
    print(json.dumps({'threads': len(bundle['threads']), 'messages': sum(len(t['messages']) for t in bundle['threads']), 'bytes': len(data), 'oversized_skipped': skipped, 'output': str(target)}, ensure_ascii=False))


if __name__ == '__main__':
    try:
        main()
    except Exception as error:
        print('Export failed: '+type(error).__name__, file=sys.stderr)
        sys.exit(1)
