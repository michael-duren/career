import concurrent.futures, hashlib, html, json, re, sys, urllib.request
from pathlib import Path
source=(Path(__file__).resolve().parents[2]/'internal/leetgrinder/readings.go').read_text()
urls=sorted(set(json.loads(x) for x in re.findall(r'URL:\s*("(?:[^"\\]|\\.)*")',source)))
cache=Path('/tmp/leetgrinder-link-check'); cache.mkdir(exist_ok=True)
previous_path=Path('/tmp/leetgrinder-links.json')
previous={r['url']:r for r in json.loads(previous_path.read_text())} if previous_path.exists() else {}
def check(url):
 if previous.get(url,{}).get('status')==200: return previous[url]
 result={'url':url}
 try:
  request=urllib.request.Request(url,headers={'User-Agent':'Mozilla/5.0 (educational bibliography link verification)'})
  with urllib.request.urlopen(request,timeout=25) as response:
   data=response.read(24*1024*1024)
   result.update(status=response.status,final_url=response.geturl(),content_type=response.headers.get('Content-Type',''),bytes=len(data))
   suffix='.pdf' if data.startswith(b'%PDF') else '.html'
   path=cache/(hashlib.sha256(url.encode()).hexdigest()[:16]+suffix)
   path.write_bytes(data); result['file']=str(path)
   if suffix=='.html':
    match=re.search(r'<title[^>]*>(.*?)</title>',data.decode('utf-8',errors='replace'),re.S|re.I)
    if match: result['title']=html.unescape(re.sub(r'\s+',' ',match[1]).strip())
 except Exception as e: result['error']=str(e)
 return result
with concurrent.futures.ThreadPoolExecutor(max_workers=6) as pool:
 results=list(pool.map(check,urls))
Path('/tmp/leetgrinder-links.json').write_text(json.dumps(results,indent=2))
for r in results:
 if 'error' in r: print('FAIL',r['url'],r['error'])
 elif re.search(r'captcha|access denied|just a moment|sign in|^404|page not found',r.get('title',''),re.I): print('CHECK',r['url'],r['title'])
print(f"Checked {len(results)} documents: {sum(r.get('status')==200 for r in results)} HTTP 200; report /tmp/leetgrinder-links.json")
