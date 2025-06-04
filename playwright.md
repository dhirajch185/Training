Playwright needs node.js as a dependency
It uses either javascript or typescript
when we run playwright init, it creates playwright.config.ts file which contains options like 'retries' 'parallel run options' etc.
example---> retries: process.env.CI ? 2: 0 means if its run using any CI like githubactions, it retries 2 times and if run locally, it runs one time.
