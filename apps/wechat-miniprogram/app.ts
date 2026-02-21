import type { FinalizeResponse } from "./types/shared";

interface GlobalData {
  userId: string;
  catId: string;
  lastResult?: FinalizeResponse;
}

App<{ globalData: GlobalData }>({
  globalData: {
    userId: "demo-user-001",
    catId: "cat-default",
  },
});
