import pathlib,json,statistics,hashlib
b=pathlib.Path('/tmp/lumen-research-20261011');base=json.loads((b/'experiment-results.json').read_text());fast=json.loads((b/'experiment-ftsfirst-results.json').read_text());orig={(r['id'],r['method']):r for r in base['bm25']['results']};diff=[];rows=[]
for r in fast['results']:
 o=orig[r['id'],r['method']]
 if o['hits']!=r['hits']:diff.append(dict(id=r['id'],method=r['method'],before=o['hits'],after=r['hits']))
 measured=next(x for x in base['rows'] if x['id']==r['id'] and x['method']==r['method'])
 rows.append(dict(id=r['id'],method=r['method'],before_ms=o['median_ms'],after_ms=r['median_ms'],after_ms_with_expansion=r['median_ms']+measured['expansion_ms']))
summary=dict(requests=len(rows),queries=len({r['id'] for r in rows}),identical_ordered_hits_and_scores=not diff,differences=diff,rows=rows,median_engine_ms_before=statistics.median(r['before_ms'] for r in rows),median_engine_ms_after=statistics.median(r['after_ms'] for r in rows),methods={m:dict(median_warm_ms=statistics.median(r['after_ms_with_expansion'] for r in rows if r['method']==m),median_engine_ms=statistics.median(r['after_ms'] for r in rows if r['method']==m),max_engine_ms=max(r['after_ms'] for r in rows if r['method']==m)) for m in ['bm25','bm25_fuzzy']})
assert all(r['payload_bytes']<=4096 and len(r['hits'])<=3 for r in base['rows'])
(b/'experiment-plan-comparison.json').write_text(json.dumps(summary,indent=2));print(json.dumps({k:v for k,v in summary.items() if k!='rows'},indent=2));print('queries_sha256',hashlib.sha256((b/'experiment-queries.json').read_bytes()).hexdigest())
