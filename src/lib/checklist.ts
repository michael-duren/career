export function checklist(body: string) {
  let fence = '';
  let section = '';
  return body.split('\n').flatMap((line, index) => {
    const marker = /^\s{0,3}(`{3,}|~{3,})/.exec(line)?.[1];
    if (marker) { if (!fence) fence = marker; else if (marker[0] === fence[0] && marker.length >= fence.length) fence = ''; return []; }
    if (fence) return [];
    const heading = /^##\s+(.+?)\s*$/.exec(line);
    if (heading) section = heading[1].toLowerCase();
    const match = /^\s*(?:[-*+]|\d+[.)])\s+\[([ xX])\]\s+(.*)$/.exec(line);
    return match ? [{ index, checked: match[1].toLowerCase() === 'x', label: match[2].trim(), section }] : [];
  });
}
export function toggleTask(body: string, index: number, checked: boolean) {
  if (!checklist(body).some(task => task.index === index)) throw new Error('Checklist item changed. Reload and try again.');
  const lines = body.split('\n');
  lines[index] = lines[index].replace(/\[[ xX]\]/, checked ? '[x]' : '[ ]');
  return lines.join('\n');
}
