import json,subprocess
from pathlib import Path
root=Path(__file__).resolve().parents[2]
rows=[]
for path in sorted((root/'internal/leetgrinder').glob('sources_*.json')):
 rows.extend(json.loads(path.read_text()))
rows.sort(key=lambda row:row['day'])
assert [row['day'] for row in rows] == list(range(1,85))
fields={'title':'Title','url':'URL','guidance':'Guidance','minutes':'Minutes','optional':'Optional','kind':'Kind','supports':'Supports'}
valid_kinds={'introduction','paper','reference'}
lines=['package leetgrinder','','// lessonReadings returns the ordered excerpts and source attribution for one lesson.','func lessonReadings(day int) []Reading {','\tswitch day {']
for row in rows:
 lines.extend([f'\tcase {row["day"]}:','\t\treturn []Reading{'])
 for reading in row['readings']:
  assert set(reading)==set(fields),reading
  assert reading['kind'] in valid_kinds,f'day {row["day"]}: unknown kind {reading["kind"]!r}'
  lines.append('\t\t\t{')
  for key,name in fields.items():
   value=json.dumps(reading[key],ensure_ascii=False)
   lines.append(f'\t\t\t\t{name}: {value},')
  lines.append('\t\t\t},')
 lines.append('\t\t}')
lines.extend(['\tdefault:','\t\treturn nil','\t}','}'])
out=root/'internal/leetgrinder/readings.go'
out.write_text('\n'.join(lines)+'\n')
subprocess.run(['gofmt','-w',str(out)],check=True)
print(f'Integrated {len(rows)} lessons with {sum(len(r["readings"]) for r in rows)} readings')
