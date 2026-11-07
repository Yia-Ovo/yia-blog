"""容器固定检查入口；复制源码后检查，不挂载生产数据库、凭据或网络。"""
import shutil
import subprocess
import sys
from pathlib import Path

workspace=Path('/workspace')
for directory in ('blog-web','blog-server'):
    shutil.copytree(Path('/source')/directory, workspace/directory, ignore=shutil.ignore_patterns('.git','node_modules','dist','.gocache','config.*.yaml'))
commands={
    'web': [('npm',['run','typecheck']),('npm',['test']),('npm',['run','build'])],
    'server': [('go',['test','-p','1','./...']),('go',['vet','-p','1','./...']),('go',['build','-p','1','-o','/tmp/blog-server','.'])],
}
kind=sys.argv[1]
cwd=workspace/('blog-web' if kind=='web' else 'blog-server')
if kind=='web':
    shutil.copytree('/opt/deps/node_modules',cwd/'node_modules',symlinks=True)
for executable,args in commands[kind]:
    print('CHECK', executable, *args, flush=True)
    result=subprocess.run([executable,*args],cwd=cwd)
    if result.returncode:
        sys.exit(result.returncode)
