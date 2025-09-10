# Playwright Automation

Dependencies: install dependencies for playwright using this command 'npm init playwright'

1) The test script files in javascript are called spec file. so, name the scripts as 'Scriptname.spec.js'
2) import playwright and assign it to an annotation called test. by using 'import {test} from @playwright/test;
3) Actual tests are written as functions and they run in async fashion. so, we need to explicitly call 'await' before each step in the function.
4) the fixtures called in the async funtion gives us ability to perform actions. like below
```
test.only("First Playwright test",async ( {page}) => {

 await page.goto( "https://rahulshettyacademy.com/client/#/auth/login" );
 console.log(await page.title());

});
```
