#!/usr/bin/env python3
import pathlib,shutil,json,re,hashlib,time,subprocess,math,statistics,collections
BASE=pathlib.Path('/tmp/lumen-research-20261011'); COPY=BASE/'experiment-checkout'; CORPUS=BASE/'experiment-corpus'; CAP=4096; K=3
# Declared before any retrieval evaluation. Source-aware deterministic synthetic cases, not independently held-out relevance judgments.
CASES=[]
def add(group,kind,q,path=None,needle=None): CASES.append(dict(id=f'{group}.{kind}.{sum(c["group"]==group and c["kind"]==kind for c in CASES)+1}',group=group,kind=kind,query=q,path=path,needle=needle))
for kind,q,p,n in [
 ('exact','refreshAccessToken','smoke/session/refresh.go','refreshAccessToken'),('exact','processPayment','smoke/billing/payment.ts','processPayment'),
 ('typo','refeshAccessToken','smoke/session/refresh.go','refreshAccessToken'),('typo','proccessPayment','smoke/billing/payment.ts','processPayment'),
 ('decomposition','refresh access token','smoke/session/refresh.go','refreshAccessToken'),('decomposition','reserve slot','smoke/billing/reservation.ts','reserveSlot'),
 ('paraphrase','stop repeated invoice charges','smoke/billing/payment.ts','processPayment'),('paraphrase','renew expired credentials','smoke/session/refresh.go','refreshAccessToken'),
 ('no_answer','quantum annealing qubit stabilizer',None,None),('no_answer','refreshAccessToken validates hardware attestation',None,None)]:add('mechanism',kind,q,p,n)
for kind,q,p,n in [
 ('exact','ValidateToken','sample/auth.go','ValidateToken'),('exact','GetEnvWithDefault','sample/config.go','GetEnvWithDefault'),('exact','BeginTransaction','sample/database.go','BeginTransaction'),('exact','HandleCreateUser','sample/handler.go','HandleCreateUser'),
 ('typo','ValidteToken','sample/auth.go','ValidateToken'),('typo','GetEnvWithDefalt','sample/config.go','GetEnvWithDefault'),('typo','BeginTransacton','sample/database.go','BeginTransaction'),('typo','HandleCreatUser','sample/handler.go','HandleCreateUser'),
 ('decomposition','validate token','sample/auth.go','ValidateToken'),('decomposition','get env with default','sample/config.go','GetEnvWithDefault'),('decomposition','begin transaction','sample/database.go','BeginTransaction'),('decomposition','handle create user','sample/handler.go','HandleCreateUser'),
 ('paraphrase','invalidate a login credential','sample/auth.go','RevokeSession'),('paraphrase','read configuration from environment variables','sample/config.go','LoadConfig'),('paraphrase','start a database transaction','sample/database.go','BeginTransaction'),('paraphrase','add a new user record','sample/database.go','InsertUser'),
 ('no_answer','GetEnvWithDefaulter',None,None),('no_answer','ValidateToken rejects revoked signing certificates',None,None),('no_answer','photosynthesis chloroplast carbon fixation',None,None),('no_answer','PostQuantumKeyExchangeNeverExists',None,None)]:add('prospective_synthetic',kind,q,p,n)
(BASE/'experiment-queries.json').write_text(json.dumps(CASES,indent=2))
for s,d in [('retrieval/corpora/smoke','smoke'),('sample-project','sample'),('fixtures','fixtures')]:shutil.copytree(COPY/'testdata'/s,CORPUS/d,dirs_exist_ok=True)
files={str(p.relative_to(CORPUS)):p.read_text(errors='replace') for p in sorted(CORPUS.rglob('*')) if p.is_file()}
manifest={p:hashlib.sha256(t.encode()).hexdigest() for p,t in files.items()};(BASE/'experiment-corpus-manifest.json').write_text(json.dumps(manifest,indent=2))
start=time.perf_counter(); vocab=collections.Counter(w.lower() for t in files.values() for w in re.findall(r'[A-Za-z_][A-Za-z_0-9]*',t) if len(w)>=5)
def grams(w):return {w[i:i+3] for i in range(len(w)-2)}
postings=collections.defaultdict(set)
for w in vocab:
 for g in grams(w):postings[g].add(w)
lex_build_ms=(time.perf_counter()-start)*1000
# Bounded Levenshtein. Character trigrams use OR candidate generation; edit distance verifies candidates.
def edit(a,b,k):
 if abs(len(a)-len(b))>k:return k+1
 prev=list(range(len(b)+1))
 for i,x in enumerate(a,1):
  row=[i]+[0]*len(b)
  for j,y in enumerate(b,1):row[j]=min(row[j-1]+1,prev[j]+1,prev[j-1]+(x!=y))
  if min(row)>k:return k+1
  prev=row
 return prev[-1]
def expand(q):
 out=[]
 for w in re.findall(r'[A-Za-z_][A-Za-z_0-9]*',q.lower()):
  if len(w)<5 or w in vocab:continue
  k=1 if len(w)<12 else 2;counts=collections.Counter(v for g in grams(w) for v in postings.get(g,()))
  candidates=[]
  shortlist=sorted(((v,n) for v,n in counts.items() if abs(len(w)-len(v))<=k),key=lambda x:(-x[1],-vocab[x[0]],x[0]))[:128]
  for v,overlap in shortlist:
   if (distance:=edit(w,v,k))<=k:candidates.append((distance,-overlap,-vocab[v],v))
  out.extend(v for _,_,_,v in sorted(candidates)[:3])
 return q+' '+ ' '.join(sorted(set(out))) if out else q
expanded={c['id']:expand(c['query']) for c in CASES}
exp_times=[];exp_by_id=collections.defaultdict(list)
for _ in range(7):
 for c in CASES:
  st=time.perf_counter();expand(c['query']);elapsed=(time.perf_counter()-st)*1000;exp_times.append(elapsed);exp_by_id[c['id']].append(elapsed)
requests=[dict(id=c['id'],method=m,query=c['query'] if m=='bm25' else expanded[c['id']]) for c in CASES for m in ['bm25','bm25_fuzzy']]
(BASE/'experiment-requests.json').write_text(json.dumps(requests,indent=2))
# Strong lexical rg baseline: case-insensitive substring OR search over original identifiers AND camel/snake components;
# score evidence windows by distinct-term IDF. No query-specific synonyms or manual substitutions.
STOP=set('a an the of for to in on with from and or where how is are does do can by at whether'.split())
def terms(q):
 raw=re.findall(r'[A-Za-z_][A-Za-z_0-9]*',q);out=[]
 for w in raw:
  out.append(w.lower());out.extend(re.sub(r'([a-z0-9])([A-Z])',r'\1 \2',w).replace('_',' ').lower().split())
 return sorted({w for w in out if len(w)>1 and w not in STOP})
def cap(hits):
 out=[]
 for h in hits[:K]:
  h=dict(h);h['content']=h['content'][:1024]
  while h['content'] and len(json.dumps(out+[h],ensure_ascii=False).encode())>CAP:h['content']=h['content'][:-64]
  if len(json.dumps(out+[h],ensure_ascii=False).encode())<=CAP:out.append(h)
 return out

def rg_search(q):
 ts=terms(q)
 cmd=['rg','--json','-i','-F','--no-ignore','--glob','!*.db']
 for w in ts:cmd+=['-e',w]
 cmd+=['.'];p=subprocess.run(cmd,cwd=CORPUS,capture_output=True,text=True)
 if p.returncode not in (0,1):raise RuntimeError(p.stderr)
 matches=collections.defaultdict(list)
 for l in p.stdout.splitlines():
  e=json.loads(l)
  if e['type']=='match':matches[e['data']['path']['text'].removeprefix('./')].append(e['data']['line_number'])
 df={w:sum(w in files[f].lower() for f in matches) for w in ts};idf={w:math.log((len(files)+1)/(df[w]+1))+1 for w in ts}
 hits=[]
 for f,ls in matches.items():
  lines=files[f].splitlines();best=None
  for line in ls:
   lo=max(0,line-5);hi=min(len(lines),line+4);content='\n'.join(lines[lo:hi]);lc=content.lower()
   score=sum(idf[w] for w in ts if w in lc)
   candidate=(score,-lo,content,lo+1)
   if best is None or candidate[:2]>best[:2]:best=candidate
  hits.append(dict(file=f,content=best[2],line=best[3],score=best[0]))
 hits.sort(key=lambda h:(-h['score'],h['file'],h['line']))
 return cap(hits)
for suffix in ['', '-wal', '-shm']:
 (BASE/('experiment-index.db'+suffix)).unlink(missing_ok=True)
raw=json.loads(subprocess.check_output([str(BASE/'experiment-bm25'),str(CORPUS),str(BASE/'experiment-index.db'),str(BASE/'experiment-requests.json')],cwd=COPY))
rows=[]
for c in CASES:
 for m in ['rg','bm25','bm25_fuzzy']:
  if m=='rg':
   times=[]
   for _ in range(5):
    st=time.perf_counter();hits=rg_search(c['query']);times.append((time.perf_counter()-st)*1000)
   query_ms=statistics.median(times)
  else:
   r=next(r for r in raw['results'] if r['id']==c['id'] and r['method']==m);hits=cap(r['hits'] or []);query_ms=r['median_ms']
  relevant=[i+1 for i,h in enumerate(hits) if c['path'] and h['file']==c['path'] and c['needle'] in h['content']]
  exp_ms=statistics.median(exp_by_id[c['id']]) if m=='bm25_fuzzy' else 0
  rows.append(dict(**c,method=m,expansion_ms=exp_ms,engine_ms=query_ms,expanded_query=expanded[c['id']] if m=='bm25_fuzzy' else c['query'],hits=hits,hit=bool(relevant),rr=1/min(relevant) if relevant else 0,query_ms=query_ms+exp_ms,payload_bytes=len(json.dumps(hits,ensure_ascii=False).encode()),relevant_count=len(relevant)))
result=dict(corpus_files=len(files),corpus_bytes=sum(len(t.encode()) for t in files.values()),lexicon_tokens=len(vocab),lexicon_build_ms=lex_build_ms,expansion_median_ms=statistics.median(exp_times),expansion_p95_ms=sorted(exp_times)[int(.95*len(exp_times))],bm25=raw,rows=rows)
result['boundary_diagnostics']={'edit_one_zero_trigram_overlap':{'query':'abxde','candidate':'abcde','levenshtein':edit('abxde','abcde',1),'common_trigrams':len(grams('abxde')&grams('abcde')),'retrievable_by_gram_candidates':False},'shorter_than_5_tokens':'fuzzy expansion intentionally disabled','candidate_cap':128,'expansion_cap_per_token':3}
assert all(r['payload_bytes']<=CAP and len(r['hits'])<=K for r in rows)
(BASE/'experiment-results.json').write_text(json.dumps(result,indent=2))
summary=[]
for group in ['mechanism','prospective_synthetic']:
 for kind in ['exact','typo','decomposition','paraphrase','no_answer','all_answerable']:
  for m in ['rg','bm25','bm25_fuzzy']:
   rs=[r for r in rows if r['group']==group and r['method']==m and (r['kind']==kind or kind=='all_answerable' and r['kind']!='no_answer')]
   summary.append(dict(group=group,kind=kind,method=m,n=len(rs),hit_at_3=sum(r['hit'] for r in rs)/len(rs),mrr=sum(r['rr'] for r in rs)/len(rs),nonempty=sum(bool(r['hits']) for r in rs)/len(rs),judged_precision=sum(r['relevant_count'] for r in rs)/max(1,sum(len(r['hits']) for r in rs)),median_ms=statistics.median(r['query_ms'] for r in rs),mean_payload_bytes=statistics.mean(r['payload_bytes'] for r in rs)))
(BASE/'experiment-summary.json').write_text(json.dumps(summary,indent=2))
print(json.dumps({k:v for k,v in result.items() if k not in ['rows','bm25']}));print('BM25 build:',raw['build_ms'],'index bytes:',raw['index_bytes'])
for r in summary:print(json.dumps(r))
