const { ensureSession } = require("./utils/api");

App({
  globalData: {
    userId: "guest",
    catId: "cat-default",
  },
  async onLaunch() {
    try {
      await ensureSession();
    } catch (err) {
      // ignore on boot; requests retry auth when needed
    }
  },
});
