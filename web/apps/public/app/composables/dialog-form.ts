import { ref } from "#imports";

import { apiErrorMessage } from "~/utils/errors";

/**
 * State wiring for a single-name dialog form: open/name/error/submitting refs
 * around a perform function. Submit trims, guards re-entry, closes and clears
 * on success, and maps failures to a user-facing message.
 */
export function useDialogForm(perform: (name: string) => Promise<void>) {
  const open = ref(false);
  const name = ref("");
  const error = ref("");
  const submitting = ref(false);

  async function submit() {
    const trimmed = name.value.trim();
    if (trimmed === "" || submitting.value) return;
    submitting.value = true;
    error.value = "";
    try {
      await perform(trimmed);
      open.value = false;
      name.value = "";
    } catch (err) {
      error.value = apiErrorMessage(err, trimmed);
    } finally {
      submitting.value = false;
    }
  }

  return { open, name, error, submitting, submit };
}
