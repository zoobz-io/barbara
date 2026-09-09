import type { Alias as IconAlias } from "#build/types/icon-sheets";

/** One toolbar button: identity for the command wiring, icon, and label. */
export type EditorTool = {
  key: string;
  icon: IconAlias;
  label: string;
};

/**
 * The WYSIWYG toolbar, in groups. Behavior attaches in the editor component
 * — a command and active-check per key, mirroring the definition/wiring
 * split the data widgets use.
 */
export const EDITOR_TOOLBAR: EditorTool[][] = [
  [
    { key: "undo", icon: "undo", label: "Undo" },
    { key: "redo", icon: "redo", label: "Redo" },
  ],
  [
    { key: "heading-1", icon: "heading-1", label: "Heading 1" },
    { key: "heading-2", icon: "heading-2", label: "Heading 2" },
    { key: "heading-3", icon: "heading-3", label: "Heading 3" },
  ],
  [
    { key: "bold", icon: "bold", label: "Bold" },
    { key: "italic", icon: "italic", label: "Italic" },
    { key: "strikethrough", icon: "strikethrough", label: "Strikethrough" },
    { key: "code", icon: "code", label: "Inline code" },
  ],
  [
    { key: "list", icon: "list", label: "Bullet list" },
    { key: "list-ordered", icon: "list-ordered", label: "Ordered list" },
    { key: "quote", icon: "quote", label: "Blockquote" },
    { key: "code-block", icon: "code-block", label: "Code block" },
    { key: "rule", icon: "minus", label: "Horizontal rule" },
  ],
];
