Playwright needs node.js as a dependency
It uses either javascript or typescript
when we run playwright init, it creates playwright.config.ts file which contains options like 'retries' 'parallel run options' etc.
example---> retries: process.env.CI ? 2: 0 means if its run using any CI like githubactions, it retries 2 times and if run locally, it runs one time.
-->   timeout: 30_000, // 30k milli seconds/or 30 sec,  globalTimeout: 10 * 60 * 1000, // 10 minutes (This says if the tests takes more than 10 mins, the test will fail)
use section in this file has more options which we can use like actionTimeout: 0, ignoreHTTPSErrors: true,trace: "on" (changed from default - on-first-retry),    video: "retain-on-failure",
    screenshot: "only-on-failure",    headless: true,     baseURL: "https://practicesoftwaretesting.com" (changed from default commented localhost string),

    
