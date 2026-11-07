import importlib.util
import os
from pathlib import Path
import unittest

spec=importlib.util.spec_from_file_location('exporter',Path(__file__).with_name('export-codex-chats.py'))
exporter=importlib.util.module_from_spec(spec)
spec.loader.exec_module(exporter)


class ExportTests(unittest.TestCase):
    def test_only_user_and_agent_text_without_context_or_credentials(self):
        secret='test-provider-value-1234567890'
        thread={'updatedAt':1794052000,'turns':[{'items':[
            {'id':'system','type':'reasoning','text':'never export internal reasoning'},
            {'id':'tool','type':'commandExecution','aggregatedOutput':secret},
            {'id':'prompt','type':'userMessage','content':[{'type':'text','text':'<environment_context>secret context</environment_context>问题 '+secret},{'type':'image','url':secret}]},
            {'id':'instructions','type':'userMessage','content':[{'type':'text','text':'# AGENTS.md instructions\n<INSTRUCTIONS>never export guide</INSTRUCTIONS>'}]},
            {'id':'reply','type':'agentMessage','text':'回答 api_key=abcdefghijklmnopqrstuvw'},
        ]}]}
        messages=exporter.text_messages(thread,[secret])
        self.assertEqual([m['role'] for m in messages],['user','assistant'])
        self.assertNotIn(secret,str(messages));self.assertNotIn('secret context',str(messages))
        self.assertNotIn('abcdefghijklmnopqrstuvw',str(messages))

    def test_workspace_checked_again_after_read(self):
        root=str(Path('yia-blog').resolve())
        class Fake:
            def call(self,method,params):
                if method=='thread/list':return {'data':[{'id':'12345678-1234-1234-1234-123456789012','cwd':root,'updatedAt':1}], 'nextCursor':None}
                return {'thread':{'cwd':str(Path('other-project').resolve()),'updatedAt':1,'turns':[]}}
        with self.assertRaises(RuntimeError):exporter.export(Fake(),{os.path.normcase(root):'yia-blog'},[],selected=[])


if __name__=='__main__':unittest.main()
