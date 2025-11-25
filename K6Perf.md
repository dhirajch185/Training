Install K6 on windows using ` winget install k6 --source winget `

Create a .js file and import the k6 libraries. ` import http from 'k6/http'; `

create a function and use http.get call to generate load on the url.

example: 
```
import http from 'k6/http';

export default function () {
   http.get("https://quickpizza.grafana.com/test.k6.io/"); 
}
```

To run the script, run the command ` k6 run ScriptName.js `
------------------------------------------------------------------------
To setup number of vuser, create a function export like
```
// User options for virtual users and duration
export const options = {
   vus: 10, // Number of virtual users
   duration: '10s', // Duration of the test
};
```

-------------------------------------------------------------------------
To capture response from the web request capture it in a variable like below
```
export default function() {

   const respLaunch = http.get("https://quickpizza.grafana.com/test.k6.io/");
   console.log(respLaunch.status);

}
```
-----------------------------------------------------------------------------
For implementing assertions in the script use a deconstruct called 'check' like this.
```
import {check} from "k6";

export default function() {

   const respLaunch = http.get("https://quickpizza.grafana.com/news.php");
   console.log(respLaunch.body);
   
   check (respLaunch, {
      'Found --> In the news' : (r) => r.body.includes("In the news")
   } );

//Here we are referencing response message from RespLaunch variable using another variable 'r' and validating if the text 'In the news' is available in the text using 'check' function.

}
```
------------------------------------------------------------------------------------

Similarly, different types of checks can be done on the same response
```
   check(respLaunch, {
      'Status is 200': (r) => r.status === 200,
      'Body contains In the news': (r) => r.body.includes('In the news'),
      'Response time is less than 500ms': (r) => r.timings.duration < 500,
      'Content-Type is text/html': (r) => r.headers['Content-Type'] === 'text/html; charset=utf-8',
      'Body length is greater than 1000 bytes': (r) => r.body.length > 1000,
      ' No server errors (5xx)': (r) => r.status < 500,
      'No client errors (4xx)': (r) => r.status < 400

   })
```

---------------------------------------------------------------------------------------
To Pass or Fail a test based on 95th percentile values or Error rate, use below commands

```
import http from "k6/http";
import {check} from "k6";
import {sleep} from "k6";

export const options = {
   vus: 1,
   duration: "10s",

   thresholds: {
      http_req_failed: ["rate<0.01"], // http errors
      http_req_duration: ["p(95)<100"] // 95% of requests must complete below 500ms
   }

};
```

