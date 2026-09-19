import { useAssetStore } from "~/stores/assets";
import { childPath } from "~/utils/path";

/**
 * Uploading into one folder, shared by the drop zone and the header's
 * upload button: each file goes to the folder under its own name, and the
 * folder's level reloads as each lands. Failures stay on the upload row.
 * Call in setup.
 */
export function useAssetUpload(appId: string, path: string) {
  const store = useAssetStore(appId);

  function send(files: FileList | File[] | undefined | null) {
    for (const file of files ?? []) {
      store
        .upload(childPath(path, file.name), file)
        .then(() => store.load(path))
        .catch(() => undefined); // the row carries the error
    }
  }

  return { send };
}
