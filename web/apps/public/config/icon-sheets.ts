import { defineNuxtIconSheetsConfig } from "@icon-sheets/nuxt/config";

// App-local aliases, merged over foundation's catalog by the layer cascade.
export default defineNuxtIconSheetsConfig({
  icons: {
    // WYSIWYG editor toolbar.
    bold: "lucide:bold",
    italic: "lucide:italic",
    strikethrough: "lucide:strikethrough",
    code: "lucide:code",
    "code-block": "lucide:square-code",
    list: "lucide:list",
    "list-ordered": "lucide:list-ordered",
    quote: "lucide:text-quote",
    "heading-1": "lucide:heading-1",
    "heading-2": "lucide:heading-2",
    "heading-3": "lucide:heading-3",
    undo: "lucide:undo-2",
    redo: "lucide:redo-2",
    save: "lucide:save",
    // Top bar section tabs.
    "file-text": "lucide:file-text",
    image: "lucide:image",
    history: "lucide:history",
    // Top bar user menu.
    sun: "lucide:sun",
    moon: "lucide:moon",
    "log-out": "lucide:log-out",
    // Asset browser file icons, by media type (plain `file` and `folder`
    // come from foundation).
    "file-image": "lucide:file-image",
    "file-code": "lucide:file-code",
    "file-spreadsheet": "lucide:file-spreadsheet",
    "file-archive": "lucide:file-archive",
    "file-video": "lucide:video",
    "file-audio": "lucide:file-audio",
    // Asset and content folder header actions, and the drop zone.
    upload: "lucide:upload",
    "folder-plus": "lucide:folder-plus",
    "file-plus": "lucide:file-plus",
    // Asset row actions.
    pencil: "lucide:pencil",
    "folder-input": "lucide:folder-input",
    "folder-up": "lucide:corner-left-up",
  },
});
