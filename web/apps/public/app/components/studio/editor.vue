<script setup lang="ts">
import type { Editor } from "@tiptap/vue-3";

import { Markdown } from "@tiptap/markdown";
import StarterKit from "@tiptap/starter-kit";
import { EditorContent, useEditor } from "@tiptap/vue-3";

import { ref } from "#imports";

import type { DocumentContent } from "~/types/documents";
import { EDITOR_TOOLBAR } from "~/constants/editor";
import { apiErrorMessage } from "~/utils/errors";
import { useDocumentStore } from "~/stores/documents";

// The page owns the document fetch (blocking useAsyncData) and remounts
// this component per document; the editor only edits and saves. Save is
// the page header's button: the state and the action are exposed to it.
const props = defineProps<{
  documentId: string;
  doc: DocumentContent | null;
  refresh: () => Promise<void>;
}>();

const store = useDocumentStore();

const dirty = ref(false);
const saving = ref(false);
const error = ref("");

// Bumped per transaction so toolbar active states re-evaluate.
const tick = ref(0);

const editor = useEditor({
  extensions: [StarterKit, Markdown],
  contentType: "markdown",
  content: props.doc?.content?.content ?? "",
  onUpdate: () => {
    dirty.value = true;
  },
  onTransaction: () => {
    tick.value++;
  },
});

async function save() {
  if (!editor.value || saving.value) return;
  saving.value = true;
  error.value = "";
  try {
    await store.saveVersion(
      props.documentId,
      props.doc?.content?.version_number ?? 0,
      editor.value.getMarkdown(),
    );
    // Resync the head version so the next save bases on it.
    await props.refresh();
    dirty.value = false;
  } catch (err) {
    error.value = apiErrorMessage(err, props.doc?.document.key ?? "");
  } finally {
    saving.value = false;
  }
}

// The toolbar's behavior half, keyed to EDITOR_TOOLBAR's descriptors.
function chain(e: Editor) {
  return e.chain().focus();
}

const commands: Record<string, (e: Editor) => void> = {
  undo: (e) => void chain(e).undo().run(),
  redo: (e) => void chain(e).redo().run(),
  "heading-1": (e) => void chain(e).toggleHeading({ level: 1 }).run(),
  "heading-2": (e) => void chain(e).toggleHeading({ level: 2 }).run(),
  "heading-3": (e) => void chain(e).toggleHeading({ level: 3 }).run(),
  bold: (e) => void chain(e).toggleBold().run(),
  italic: (e) => void chain(e).toggleItalic().run(),
  strikethrough: (e) => void chain(e).toggleStrike().run(),
  code: (e) => void chain(e).toggleCode().run(),
  list: (e) => void chain(e).toggleBulletList().run(),
  "list-ordered": (e) => void chain(e).toggleOrderedList().run(),
  quote: (e) => void chain(e).toggleBlockquote().run(),
  "code-block": (e) => void chain(e).toggleCodeBlock().run(),
  rule: (e) => void chain(e).setHorizontalRule().run(),
};

const activeChecks: Record<string, (e: Editor) => boolean> = {
  "heading-1": (e) => e.isActive("heading", { level: 1 }),
  "heading-2": (e) => e.isActive("heading", { level: 2 }),
  "heading-3": (e) => e.isActive("heading", { level: 3 }),
  bold: (e) => e.isActive("bold"),
  italic: (e) => e.isActive("italic"),
  strikethrough: (e) => e.isActive("strike"),
  code: (e) => e.isActive("code"),
  list: (e) => e.isActive("bulletList"),
  "list-ordered": (e) => e.isActive("orderedList"),
  quote: (e) => e.isActive("blockquote"),
  "code-block": (e) => e.isActive("codeBlock"),
};

function runTool(key: string) {
  if (editor.value) commands[key]?.(editor.value);
}

function isToolActive(key: string): boolean {
  void tick.value;
  return editor.value ? (activeChecks[key]?.(editor.value) ?? false) : false;
}

defineExpose({ dirty, saving, error, save });
</script>

<template>
  <div class="editor-pane">
    <div class="editor-toolbar" role="toolbar" aria-label="Formatting">
      <template v-for="(group, i) in EDITOR_TOOLBAR" :key="i">
        <span v-if="i > 0" class="toolbar-sep" />
        <button
          v-for="tool in group"
          :key="tool.key"
          type="button"
          class="toolbar-btn"
          :class="{ active: isToolActive(tool.key) }"
          :aria-label="tool.label"
          :title="tool.label"
          @click="runTool(tool.key)"
        >
          <Icon class="f-icon" fill="currentColor" :name="tool.icon" />
        </button>
      </template>
    </div>

    <div class="editor-body">
      <div class="editor-container">
        <EditorContent :editor="editor" />
      </div>
    </div>
  </div>
</template>
