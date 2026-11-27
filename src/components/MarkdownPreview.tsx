import Markdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import rehypeRaw from 'rehype-raw';
import rehypeSanitize, { defaultSchema } from 'rehype-sanitize';

// Preserve the reference pages' layout classes without allowing scripts, events, or unsafe URLs.
const schema = { ...defaultSchema, attributes: { ...defaultSchema.attributes, '*': [...(defaultSchema.attributes?.['*'] ?? []), 'className'] } };

/** Markdown drawn in the Architect tokens (the .prose rules in src/styles/global.css). */
export default function MarkdownPreview({ body }: { body: string }) {
  return <div className="prose break-words">
    <Markdown remarkPlugins={[remarkGfm]} rehypePlugins={[rehypeRaw, [rehypeSanitize, schema]]}>{body || '*Nothing to preview yet.*'}</Markdown>
  </div>;
}
