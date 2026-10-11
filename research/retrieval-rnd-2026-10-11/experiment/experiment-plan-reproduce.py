import pathlib,subprocess,os
b=pathlib.Path('/tmp/lumen-research-20261011');root=b/'experiment-checkout';p=root/'internal/store/lexical.go';original=p.read_text();old='FROM project_files pf ` + lexicalResultJoins + ` JOIN lexical_chunks ON lexical_chunks.rowid=cd.id WHERE lexical_chunks MATCH ? AND pf.project_id=?';new='FROM lexical_chunks CROSS JOIN chunk_defs cd ON lexical_chunks.rowid=cd.id CROSS JOIN project_files pf ON cd.file_revision_id=pf.file_revision_id CROSS JOIN file_revisions fr ON fr.id=pf.file_revision_id WHERE lexical_chunks MATCH ? AND pf.project_id=?';assert original.count(old)==1
try:
 p.write_text(original.replace(old,new))
 env=dict(os.environ,GOCACHE=str(b/'experiment-gocache'))
 subprocess.run(['go','build','-tags','fts5','-o',str(b/'experiment-bm25-ftsfirst'),'./cmd/experiment'],cwd=root,env=env,check=True)
finally:p.write_text(original)
with (b/'experiment-ftsfirst-results.json').open('w') as out, (b/'experiment-ftsfirst-progress.log').open('w') as err:
 subprocess.run([str(b/'experiment-bm25-ftsfirst'),str(b/'experiment-corpus'),str(b/'experiment-index.db'),str(b/'experiment-requests.json')],stdout=out,stderr=err,check=True)
subprocess.run(['python3',str(b/'experiment-compare.py')],check=True)
