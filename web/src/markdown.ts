import MarkdownIt from "markdown-it";

// html:false escapes any raw HTML in the source, so rendering ClickUp-authored
// descriptions can't inject markup. Image refs (![](url)) still become <img>.
// breaks:true keeps single newlines as line breaks, matching the old
// pre-wrap plain-text feel.
const md = new MarkdownIt({ html: false, linkify: true, breaks: true });

// Render links in a new tab so clicking one doesn't navigate away from the app.
const defaultLinkOpen =
  md.renderer.rules.link_open ||
  ((tokens, idx, options, _env, self) => self.renderToken(tokens, idx, options));
md.renderer.rules.link_open = (tokens, idx, options, env, self) => {
  tokens[idx].attrSet("target", "_blank");
  tokens[idx].attrSet("rel", "noopener noreferrer");
  return defaultLinkOpen(tokens, idx, options, env, self);
};

export function renderMarkdown(src: string): string {
  return md.render(src || "");
}
