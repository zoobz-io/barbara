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
  },
});
