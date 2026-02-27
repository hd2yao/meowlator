"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
const api_1 = require("./utils/api");
App({
    globalData: {
        userId: "guest",
        catId: "cat-default",
    },
    async onLaunch() {
        try {
            await (0, api_1.ensureSession)();
        }
        catch (err) {
            // Keep app usable in local dev; requests will retry auth lazily.
        }
    },
});
