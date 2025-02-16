// @ts-check

/** @type {import('@playwright/test').PlaywrightTestConfig} */
const config = {
    use: {
        headless: true,
        viewport: { width: 1280, height: 720 },
        actionTimeout: 5000,
    },
    reporter: 'list',
    workers: 1,
    fullyParallel: false,
    projects: [
        {
            name: 'chromium',
            use: { browserName: 'chromium' },
        },
    ],
};

export default config;
