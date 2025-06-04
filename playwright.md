Playwright needs node.js as a dependency
It uses either javascript or typescript

when we run playwright init, it creates playwright.config.ts file which contains options like 'retries' 'parallel run options' etc.
example---> retries: process.env.CI ? 2: 0 means if its run using any CI like githubactions, it retries 2 times and if run locally, it runs one time.

-->   timeout: 30_000, // 30k milli seconds/or 30 sec,  globalTimeout: 10 * 60 * 1000, // 10 minutes (This says if the tests takes more than 10 mins, the test will fail)
use section in this file has more options which we can use like actionTimeout: 0, ignoreHTTPSErrors: true,trace: "on" (changed from default - on-first-retry),    video: "retain-on-failure",
    screenshot: "only-on-failure",    headless: true,     baseURL: "https://practicesoftwaretesting.com" (changed from default commented localhost string),

Playwright gives some browser versions out of the box, but the drawback is you are limited to using those versions with the version of playwright. source: https://playwright.dev/docs/release-notes#browser-versions

Steps to create a project with a setup dependency. to ensure project always run and passes before running any other projects. Add the code to projects array. Playwright looks at this and understand how to match filenames as test for the project. We will be naming our setup files as name.setup.ts instead of name.spec.ts (default)
    {
      name: "setup",
      testMatch: /.*\.setup\.ts/,
    }
Add this option --> dependencies: ['setup'], to othe projects like chromium and firefox, webkit as well between name and use lines. this makes sure before each of the projects run, these dependencies 'setup' name run successfully before those projects run.

Also add clipboard read for chrome option like ---> use: { ...devices['Desktop Chrome'], Permissions: ['clipboard-read'] }

If we want to run the tests in headed mode though the config was setup as headless, just use ---> option cli headed in the terminal example: npx playwright test --headed

other way is to run with project name: npx playwright test --project chromium.
we can add on to it and cutomize as per our requirement. like --> npx playwright test --project chromium --project firefox
even we can run it with arguments like using regex --> npx playwright test --project "*omium"

commands to make it even easier when the playwright project grows is --> narrow down runs by running all the tests within a specific file. like --> npx playwright test tests/example.spec.ts
***--> One more option is to run the test from a specific line like--> npx playwright test tests/example.spec.ts:10

***--> To merge the tests togethe from different sections, we can use tag function like--> in this test('get started link', async ({ page }) => {
  await page.goto('https://playwright.dev/');--------> Add--> test("get started link", { tag: ["@first"] }, async ({ page }) => {
  await page.goto("https://playwright.dev/");
  Then do ---> npx playwright test --grep "@first"
  

