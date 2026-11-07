import importlib.util
import os
import tempfile
import unittest
from pathlib import Path

spec=importlib.util.spec_from_file_location('worker',Path(__file__).with_name('worker.py'))
worker=importlib.util.module_from_spec(spec);spec.loader.exec_module(worker)

class FileSafetyTests(unittest.TestCase):
    def test_rejects_sensitive_paths_and_traversal(self):
        for name in ['.private/a.md','blog-server/config/config.prod.yaml','deploy/ubuntu/install.sh','blog-web/src/../../etc/a.ts','/etc/a.ts','blog-web/src/.env','blog-web/src/../../.git/config','blog-web/src/a\\b.ts']:
            self.assertFalse(worker.allowed_file(name),name)
        self.assertTrue(worker.allowed_file('blog-web/src/views/AIView.tsx'))

    def test_unique_edit_and_new_file(self):
        with tempfile.TemporaryDirectory() as d:
            tools=worker.FileTools(Path(d))
            tools.call('edit_file',{'path':'blog-web/src/a.ts','old':'','new':'one one'})
            with self.assertRaises(ValueError):tools.call('edit_file',{'path':'blog-web/src/a.ts','old':'one','new':'two'})
            tools.call('edit_file',{'path':'blog-web/src/a.ts','old':'one one','new':'two'})
            self.assertIn('two',tools.call('read_file',{'path':'blog-web/src/a.ts','start':1,'count':1}))

    def test_no_arbitrary_commands(self):
        with tempfile.TemporaryDirectory() as d:
            tools=worker.FileTools(Path(d),lambda _: (0,'ok'))
            with self.assertRaises(ValueError):tools.call('check',{'kind':'server; curl example.com'})
            with self.assertRaises(ValueError):tools.call('exec',{'command':'echo secret'})

    def test_symlink_cannot_escape(self):
        if os.name=='nt':self.skipTest('Linux server tests verify real symlinks')
        with tempfile.TemporaryDirectory() as d, tempfile.TemporaryDirectory() as outside:
            root=Path(d);(root/'blog-web/src').mkdir(parents=True)
            (root/'blog-web/src/link').symlink_to(outside,target_is_directory=True)
            with self.assertRaises(ValueError):worker.source_path(root,'blog-web/src/link/a.ts')

    def test_general_task_cannot_inject_private_context(self):
        with tempfile.TemporaryDirectory() as d:
            with self.assertRaises(ValueError):worker.generate(Path(d),{'context_mode':'general','context':'secret','prompt':'hello'},lambda _: [])

    def test_provider_key_is_redacted(self):
        previous=os.environ.get('YIA_AI_API_KEY')
        os.environ['YIA_AI_API_KEY']='test-secret-provider-value'
        try:self.assertNotIn('test-secret-provider-value',worker.redact('a test-secret-provider-value b sk-1234567890123456789012'))
        finally:
            if previous is None:os.environ.pop('YIA_AI_API_KEY',None)
            else:os.environ['YIA_AI_API_KEY']=previous

if __name__=='__main__':unittest.main()
