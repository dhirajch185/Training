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
For implementing assertions in the script use a deconstruct called check like this.
```
