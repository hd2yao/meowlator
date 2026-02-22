import type { FinalizeResponse } from "./types/shared";
import { ensureSession } from "./utils/api";

interface GlobalData {
  userId: string;
  catId: string;
  sessionToken?: string;
  sessionExpiresAt?: number;
  authPromise?: Promise<void>;
  lastResult?: FinalizeResponse;
}

App<{ globalData: GlobalData }>({
  globalData: {
    userId: "guest",
    catId: "cat-default",
  },
  async onLaunch() {
    try {
      await ensureSession();
    } catch (err) {
      // Keep app usable in local dev; requests will retry auth lazily.
    }
  },
});
