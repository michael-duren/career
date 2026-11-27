import Markdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import rehypeRaw from 'rehype-raw';
import rehypeSanitize, { defaultSchema } from 'rehype-sanitize';

// Preserve the reference pages' layout classes without allowing scripts, events, or unsafe URLs.
const schema = { ...defaultSchema, attributes: { ...defaultSchema.attributes, '*': [...(defaultSchema.attributes?.['*'] ?? []), 'className'] } };

/** ink draws the Markdown in the Architect tokens instead of the older zinc prose. */
export default function MarkdownPreview({ body, ink = false }: { body: string; ink?: boolean }) {
  return <div className={`prose prose-invert max-w-none break-words prose-pre:overflow-x-auto${ink ? ' prose-ink' : ''}`}>
    <Markdown remarkPlugins={[remarkGfm]} rehypePlugins={[rehypeRaw, [rehypeSanitize, schema]]}>{body || '*Nothing to preview yet.*'}</Markdown>
  </div>;
}
